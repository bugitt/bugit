package ci

import (
	"strconv"
	"time"

	json "github.com/json-iterator/go"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/kube"
	log "unknwon.dev/clog/v2"
)

func deploy(ctx *Context) (err error) {
	err = ctx.updateStage(db.DeployStart, -1)
	if err != nil {
		return
	}

	config := ctx.config.Deploy

	// 双重检查是否需要启动Deploy
	checkNeedDeploy := func() bool {
		if config == nil || len(config.On) <= 0 {
			return false
		}
		for _, branch := range config.On {
			if branch == ctx.refName {
				return true
			}
		}
		return false
	}
	if !checkNeedDeploy() {
		return nil
	}

	if err = doDeploy(ctx, config); err != nil {
		return
	}

	return ctx.updateStage(db.DeployEnd, -1)
}

func doDeploy(ctx *Context, config *db.DeployTaskConfig) (err error) {
	deployName := ctx.repo.LowerName
	var (
		outputLog      string
		begin          = time.Now()
		namespace      = ctx.repo.Owner.KSProjectName
		svcName        = deployName
		deploymentName = deployName
		ip             = NextIP()
		labels         = map[string]string{
			"repo": ctx.repo.LowerName,
		}
		extraLabels = map[string]string{
			"branch":     ctx.refName,
			"commit":     ctx.commit,
			"pusher":     ctx.pusher.LowerName,
			"pusherID":   strconv.FormatInt(ctx.pusher.ID, 10),
			"pipelineID": strconv.FormatInt(ctx.pipeline.ID, 10),
		}
		svcPorts []kube.SvcPort
		result   = db.DeployResult{
			IP:             ip,
			RepoID:         ctx.repo.ID,
			Branch:         ctx.refName,
			Commit:         ctx.commit,
			Namespace:      namespace,
			ServiceName:    svcName,
			DeploymentName: deploymentName,
			BasicTaskResult: db.BasicTaskResult{
				PipelineID: ctx.pipeline.ID,
				Name:       "deploy",
				Describe:   "deploy",
			},
		}
	)

	defer func() {
		result.End(begin, err, outputLog)
		svcPortsBytes, _ := json.Marshal(svcPorts)
		result.Ports = string(svcPortsBytes)
		dbErr := db.SaveCIResult(result)
		if dbErr != nil {
			if err != nil {
				log.Error("save deploy result failed, error message: %s", dbErr.Error())
				return
			}
			err = dbErr
		}
	}()

	cli, err := kube.NewClient(ctx, &kube.CreateClientOpt{Namespace: namespace})
	if err != nil {
		return
	}

	err = cli.EnsureNS(kube.Quota{
		CPU:    config.CPU,
		Memory: config.Memory,
	})
	if err != nil {
		return
	}

	err = cli.CreateOrUpdateDockerRegistrySecret(ctx, &kube.PrivateDockerRegistrySecret{
		Name:     namespace,
		Username: conf.Harbor.AdminName,
		Password: conf.Harbor.AdminPasswd,
		Host:     conf.Harbor.Host,
	})
	if err != nil {
		return
	}

	ports := config.Ports.KubePorts()
	err = cli.PodDeploy(ctx, &kube.PodDeployOpt{
		Labels:               labels,
		ExtraLabels:          extraLabels,
		ReplicaNum:           1,
		Stateful:             config.Stateful,
		Duration:             kube.DefaultDuration,
		DockerRegistrySecret: namespace,
		Spec: kube.PodSpec{
			Name:     deployName,
			ImageTag: ctx.imageTag[0],
			Envs:     config.Envs,
			Ports:    ports,
			WorkDir:  config.WorkDir,
			Cmd: kube.Cmd{
				Command: config.Cmd.Command,
				Args:    config.Cmd.Args,
			},
			PullPolicy: kube.PullAlways,
			Quota: kube.Quota{
				CPU:    config.CPU,
				Memory: config.Memory,
			},
		},
	})
	if err != nil {
		return
	}

	svcPorts, err = cli.PodExportNodePort(ctx, &kube.PodExportNodePortOpt{
		BaseServiceOpt: kube.BaseServiceOpt{
			Name:   svcName,
			Labels: labels,
			Ports:  ports,
		},
	})
	return
}

// NextIP 获取这次应该部署的到哪个IP上（手动负载均衡）
var NextIP = func() func() string {
	i := -1
	return func() string {
		i = (i + 1) % len(conf.Devops.KubeIP)
		return conf.Devops.KubeIP[i]
	}
}()
