package db

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	log "unknwon.dev/clog/v2"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/loheagn/cloud/docker"
)

type DockerError struct {
	Error       string                 `json:"error"`
	ErrorDetail DockerBuildErrorDetail `json:"errorDetail"`
}

type DockerBuildErrorDetail struct {
	Message string `json:"message"`
}

func getDockerCli() (*client.Client, error) {
	return docker.GetClient(&docker.InitOption{
		Host: conf.Docker.DockerService,
	})
}

func BuildImage(dockerFilePath, contextPath string, tags []string) (sourceLog string, isSuccessful bool, buildErr DockerError, err error) {
	cli, err := getDockerCli()
	if err != nil {
		return
	}
	ctx := context.Background()

	buildOpts := types.ImageBuildOptions{
		Dockerfile: dockerFilePath,
		Tags:       tags,
	}

	buildCtx, _ := archive.TarWithOptions(contextPath, &archive.TarOptions{})

	resp, err := cli.ImageBuild(ctx, buildCtx, buildOpts)
	if err != nil {
		log.Error("image build error - %s", err)
	}
	defer resp.Body.Close()

	// 处理输出
	var output bytes.Buffer
	buildErr, err = readDockerOutput(resp.Body, &output)
	if err != nil {
		return
	}
	sourceLog = output.String()
	if buildErr.Error == "" {
		isSuccessful = true
	}

	log.Info(sourceLog, isSuccessful, err)

	return
}

func PushImage(tag string) (sourceLog string, isSuccessful bool, buildErr DockerError, err error) {
	cli, err := getDockerCli()
	if err != nil {
		return
	}
	ctx := context.Background()

	// just make docker client happy
	user := "root"
	password := "11111111"

	authConfig := types.AuthConfig{Username: user, Password: password}
	encodedJSON, _ := json.Marshal(authConfig)
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	resp, err := cli.ImagePush(ctx, tag, types.ImagePushOptions{
		All:           false,
		RegistryAuth:  authStr,
		PrivilegeFunc: nil,
	})
	log.Info("image tag: %s", tag)
	if err != nil {
		log.Error("image push error - %s", err.Error())
	}
	defer func() {
		if resp != nil {
			resp.Close()
		}
	}()

	// 处理输出
	var output bytes.Buffer
	buildErr, err = readDockerOutput(resp, &output)
	if err != nil {
		return
	}
	sourceLog = output.String()
	if buildErr.Error == "" {
		isSuccessful = true
	}

	return
}

func readDockerOutput(rd io.Reader, output io.Writer) (buildErr DockerError, err error) {
	var lastLine string
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		_, _ = output.Write(scanner.Bytes())
		_, _ = output.Write([]byte{'\n'})
		log.Info(string(scanner.Bytes()))
	}

	buildErr = DockerError{}
	_ = json.Unmarshal([]byte(lastLine), &buildErr)
	if err = scanner.Err(); err != nil {
		return
	}
	return
}
