// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package context

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-macaron/csrf"
	"github.com/go-macaron/session"
	"github.com/gomodule/redigo/redis"
	gouuid "github.com/satori/go.uuid"
	"gopkg.in/macaron.v1"
	log "unknwon.dev/clog/v2"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/auth"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/tool"
)

type ToggleOptions struct {
	SignInRequired  bool
	SignOutRequired bool
	AdminRequired   bool
	DisableCSRF     bool
}

func Toggle(options *ToggleOptions) macaron.Handler {
	return func(c *Context) {
		// Cannot view any page before installation.
		if !conf.Security.InstallLock {
			c.RedirectSubpath("/install")
			return
		}

		// Check prohibit login users.
		if c.IsLogged && c.User.ProhibitLogin {
			c.Data["Title"] = c.Tr("auth.prohibit_login")
			c.Success("user/auth/prohibit_login")
			return
		}

		// Check non-logged users landing page.
		if !c.IsLogged && c.Req.RequestURI == "/" && conf.Server.LandingURL != "/" {
			c.RedirectSubpath(conf.Server.LandingURL)
			return
		}

		// Redirect to dashboard if user tries to visit any non-login page.
		if options.SignOutRequired && c.IsLogged && c.Req.RequestURI != "/" {
			c.RedirectSubpath("/")
			return
		}

		if !options.SignOutRequired && !options.DisableCSRF && c.Req.Method == "POST" && !isAPIPath(c.Req.URL.Path) {
			csrf.Validate(c.Context, c.csrf)
			if c.Written() {
				return
			}
		}

		if options.SignInRequired {
			if !c.IsLogged {
				// Restrict API calls with error message.
				if isAPIPath(c.Req.URL.Path) {
					c.JSON(http.StatusForbidden, map[string]string{
						"message": "Only authenticated user is allowed to call APIs.",
					})
					return
				}

				c.SetCookie("redirect_to", url.QueryEscape(conf.Server.Subpath+c.Req.RequestURI), 0, conf.Server.Subpath)
				c.RedirectSubpath("/user/login")
				return
			} else if !c.User.IsActive && conf.Auth.RequireEmailConfirmation {
				c.Title("auth.active_your_account")
				c.Success("user/auth/activate")
				return
			}
		}

		// Redirect to log in page if auto-signin info is provided and has not signed in.
		if !options.SignOutRequired && !c.IsLogged && !isAPIPath(c.Req.URL.Path) &&
			len(c.GetCookie(conf.Security.CookieUsername)) > 0 {
			c.SetCookie("redirect_to", url.QueryEscape(conf.Server.Subpath+c.Req.RequestURI), 0, conf.Server.Subpath)
			c.RedirectSubpath("/user/login")
			return
		}

		if options.AdminRequired {
			if !c.User.IsAdmin {
				c.Status(http.StatusForbidden)
				return
			}
			c.PageIs("Admin")
		}
	}
}

func isAPIPath(url string) bool {
	return strings.HasPrefix(url, "/api/")
}

var rePool *redis.Pool

func initRedis() *redis.Pool {
	host := conf.CloudAPI.RedisHost

	rePool = &redis.Pool{
		MaxIdle:     30,
		MaxActive:   1024,
		IdleTimeout: 300,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", host)
			if err != nil {
				log.Error(err.Error())
				return nil, err
			}
			if _, err := c.Do("AUTH", "@buaa21"); err != nil {
				c.Close()
				return nil, err
			}
			if _, err = c.Do("PING"); err != nil {
				c.Close()
				log.Error(err.Error())
				return nil, err
			}
			return c, err
		},
	}
	log.Info("redis init success")
	return rePool
}

func redisAuthUserID(token string) (_ int64, isTokenAuth bool) {
	if conf.CloudAPI.SupperDebug {
		return 1, true
	}
	if rePool == nil {
		rePool = initRedis()
	}
	RedisConn := rePool.Get()
	defer RedisConn.Close()
	studentID, err := redis.String(RedisConn.Do("GET", token))
	if err != nil {
		log.Error(err.Error())
		return 0, false
	}
	if len(studentID) <= 0 {
		return 0, false
	}
	studentID = strings.Trim(studentID, "\"")
	user, err := db.GetUserByStudentID(studentID)
	if err != nil {
		log.Error(err.Error())
		return 0, false
	}
	if user == nil {
		log.Error("student user %s not found", studentID)
		return 0, false
	}
	return user.ID, true
}

// authenticatedUserID returns the ID of the authenticated user, along with a bool value
// which indicates whether the user uses token authentication.
func authenticatedUserID(c *macaron.Context, sess session.Store) (_ int64, isTokenAuth bool) {
	if !db.HasEngine {
		return 0, false
	}

	// Check access token.
	if isAPIPath(c.Req.URL.Path) {
		tokenSHA := c.Query("token")
		if len(tokenSHA) <= 0 {
			tokenSHA = c.Query("access_token")
		}
		if len(tokenSHA) == 0 {
			// Well, check with header again.
			auHead := c.Req.Header.Get("Authorization")
			if len(auHead) > 0 {
				auths := strings.Fields(auHead)
				if len(auths) == 2 && auths[0] == "token" {
					tokenSHA = auths[1]
				} else if len(auths) == 1 {
					// 从Redis中获取权限校验信息
					return redisAuthUserID(auths[0])
				}
			}
		}

		// Let's see if token is valid.
		if len(tokenSHA) > 0 {
			t, err := db.AccessTokens.GetBySHA(tokenSHA)
			if err != nil {
				if !db.IsErrAccessTokenNotExist(err) {
					log.Error("GetAccessTokenBySHA: %v", err)
				}
				return 0, false
			}
			if err = db.AccessTokens.Save(t); err != nil {
				log.Error("UpdateAccessToken: %v", err)
			}
			return t.UserID, true
		}
	}

	uid := sess.Get("uid")
	if uid == nil {
		return 0, false
	}
	if id, ok := uid.(int64); ok {
		if _, err := db.GetUserByID(id); err != nil {
			if !db.IsErrUserNotExist(err) {
				log.Error("Failed to get user by ID: %v", err)
			}
			return 0, false
		}
		return id, false
	}
	return 0, false
}

// authenticatedUser returns the user object of the authenticated user, along with two bool values
// which indicate whether the user uses HTTP Basic Authentication or token authentication respectively.
func authenticatedUser(ctx *macaron.Context, sess session.Store) (_ *db.User, isBasicAuth bool, isTokenAuth bool) {
	if !db.HasEngine {
		return nil, false, false
	}

	uid, isTokenAuth := authenticatedUserID(ctx, sess)

	if uid <= 0 {
		if conf.Auth.EnableReverseProxyAuthentication {
			webAuthUser := ctx.Req.Header.Get(conf.Auth.ReverseProxyAuthenticationHeader)
			if len(webAuthUser) > 0 {
				u, err := db.GetUserByName(webAuthUser)
				if err != nil {
					if !db.IsErrUserNotExist(err) {
						log.Error("Failed to get user by name: %v", err)
						return nil, false, false
					}

					// Check if enabled auto-registration.
					if conf.Auth.EnableReverseProxyAutoRegistration {
						u := &db.User{
							Name:     webAuthUser,
							Email:    gouuid.NewV4().String() + "@localhost",
							Passwd:   webAuthUser,
							IsActive: true,
						}
						if err = db.CreateUser(u); err != nil {
							// FIXME: should I create a system notice?
							log.Error("Failed to create user: %v", err)
							return nil, false, false
						} else {
							return u, false, false
						}
					}
				}
				return u, false, false
			}
		}

		// Check with basic auth.
		baHead := ctx.Req.Header.Get("Authorization")
		if len(baHead) > 0 {
			auths := strings.Fields(baHead)
			if len(auths) == 2 && auths[0] == "Basic" {
				uname, passwd, _ := tool.BasicAuthDecode(auths[1])

				u, err := db.Users.Authenticate(uname, passwd, -1)
				if err != nil {
					if !auth.IsErrBadCredentials(err) {
						log.Error("Failed to authenticate user: %v", err)
					}
					return nil, false, false
				}

				return u, true, false
			}
		}
		return nil, false, false
	}

	u, err := db.GetUserByID(uid)
	if err != nil {
		log.Error("GetUserByID: %v", err)
		return nil, false, false
	}
	return u, false, isTokenAuth
}
