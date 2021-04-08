// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package user

import (
	"net/http"

	api "github.com/gogs/go-gogs-client"
	"github.com/pkg/errors"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/route/api/v1/convert"
)

func ListEmails(c *context.APIContext) {
	emails, err := db.GetEmailAddresses(c.User.ID)
	if err != nil {
		c.Error(err, "get email addresses")
		return
	}
	apiEmails := make([]*api.Email, len(emails))
	for i := range emails {
		apiEmails[i] = convert.ToEmail(emails[i])
	}
	c.JSONSuccess(&apiEmails)
}

func AddEmail(c *context.APIContext, form api.CreateEmailOption) {
	if len(form.Emails) == 0 {
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	emails := make([]*db.EmailAddress, len(form.Emails))
	for i := range form.Emails {
		emails[i] = &db.EmailAddress{
			UID:         c.User.ID,
			Email:       form.Emails[i],
			IsActivated: !conf.Auth.RequireEmailConfirmation,
		}
	}

	if err := db.AddEmailAddresses(emails); err != nil {
		if db.IsErrEmailAlreadyUsed(err) {
			c.ErrorStatus(http.StatusUnprocessableEntity, errors.New("email address has been used: "+err.(db.ErrEmailAlreadyUsed).Email()))
		} else {
			c.Error(err, "add email addresses")
		}
		return
	}

	apiEmails := make([]*api.Email, len(emails))
	for i := range emails {
		apiEmails[i] = convert.ToEmail(emails[i])
	}
	c.JSON(http.StatusCreated, &apiEmails)
}

func DeleteEmail(c *context.APIContext, form api.CreateEmailOption) {
	if len(form.Emails) == 0 {
		c.NoContent()
		return
	}

	emails := make([]*db.EmailAddress, len(form.Emails))
	for i := range form.Emails {
		emails[i] = &db.EmailAddress{
			UID:   c.User.ID,
			Email: form.Emails[i],
		}
	}

	if err := db.DeleteEmailAddresses(emails); err != nil {
		c.Error(err, "delete email addresses")
		return
	}
	c.NoContent()
}
