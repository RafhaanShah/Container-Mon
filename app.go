package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containrrr/shoutrrr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/robfig/cron/v3"
)

const (
	containerLabel            = "containermon.enable"
	senderSplitter            = "|"
	envFailLimit              = "CONTAINERMON_FAIL_LIMIT"
	envCronSchedule           = "CONTAINERMON_CRON"
	envNotificationURL        = "CONTAINERMON_NOTIFICATION_URL"
	envUseLabels              = "CONTAINERMON_USE_LABELS"
	envNotifyWhenHealthy      = "CONTAINERMON_NOTIFY_HEALTHY"
	envCheckStoppedContainers = "CONTAINERMON_CHECK_STOPPED"
	envMessagePrefix          = "CONTAINERMON_MESSAGE_PREFIX"
	envCheckContainerExitCode = "CONTAINERMON_CHECK_EXIT_CODE"
)

type config struct {
	failLimit              int
	cronSchedule           string
	notificationURL        string
	useLabels              bool
	notifyWhenHealthy      bool
	checkStoppedContainers bool
	messagePrefix          string
	checkContainerExitCode bool
}

func main() {
	fmt.Println("Starting up Container-Mon")

	conf := getConfig()
	cli := getDockerClient()
	ctx := context.Background()
	cMap := make(map[string]int)

	cr := cron.New()
	cr.AddFunc(conf.cronSchedule, func() {
		err := checkContainers(ctx, cli, conf, cMap)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error checking containers: %v", err))
		}
	})

	cr.Run()
}

func checkContainers(ctx context.Context, cli *client.Client, conf config, cMap map[string]int) error {
	// Get the list of containers
	cList, err := getContainers(ctx, cli, conf.useLabels, conf.checkStoppedContainers)

	if err != nil {
		return err
	}

	// If the map has containers that are not in the new list, remove them
	for cID := range cMap {
		if !containerInList(cID, cList) {
			delete(cMap, cID)
		}
	}

	// If the list has containers that are not in the map, add them with a fail count of 0
	for _, c := range cList {
		if _, ok := cMap[c.ID]; !ok {
			cMap[c.ID] = 0
		}
	}

	// Loop over the list of containers
	for _, c := range cList {
		// If the container is healthy, set the fail count to 0
		if isHealthy(ctx, cli, c, conf) {
			// If it was previously unhealthy, notify that it is now healthy
			if conf.notifyWhenHealthy && cMap[c.ID] < 0 {
				notify(conf.notificationURL, c.Names[0], true, conf.messagePrefix)
			}
			cMap[c.ID] = 0
		} else {
			// If the container is not healthy and we have not yet sent a notification for it, add +1 to it's fail count
			if cMap[c.ID] > -1 {
				count := cMap[c.ID] + 1
				cMap[c.ID] = count
				// If the fail count has reached the max count, send a notification and set the count to -1
				if count >= conf.failLimit {
					cMap[c.ID] = -1
					notify(conf.notificationURL, c.Names[0], false, conf.messagePrefix)
				}
			}
		}
	}

	return nil
}

func containerInList(a string, list []types.Container) bool {
	for _, b := range list {
		if b.ID == a {
			return true
		}
	}

	return false
}

func getContainers(ctx context.Context, cli *client.Client, filterByLabel bool, checkStoppedContainers bool) ([]types.Container, error) {
	args := filters.NewArgs()
	if filterByLabel {
		args.Add("label", fmt.Sprintf("%v=true", containerLabel))
	}

	return cli.ContainerList(ctx, container.ListOptions{
		All:     checkStoppedContainers,
		Filters: args,
	})
}

func isHealthy(ctx context.Context, cli *client.Client, c types.Container, conf config) bool {
	running := c.State == "running"
	
	containerJSON, err := cli.ContainerInspect(ctx, c.ID)
	if conf.checkContainerExitCode && c.State == "exited" {
		// If the container is stopped ("exited"), we need to inspect it to get the exit code.
		if err != nil {
			// If we can't inspect an exited container, we can't determine its health definitively,
			// so we'll treat it as unhealthy as a precaution.
			return false
		}
		// A stopped container is considered healthy only if its exit code is 0.
		return containerJSON.State.ExitCode == 0
	} else if err != nil {
		return running
	}

	health := containerJSON.State.Health
	if health == nil || health.Status == types.NoHealthcheck || health.Status == types.Starting {
		return running
	}

	healthy := health.Status == types.Healthy
	return healthy 	
}

func getConfig() config {
	c := config{
		failLimit:              getEnvInt(envFailLimit, 1),
		cronSchedule:           getEnv(envCronSchedule, "*/5 * * * *", true),
		notificationURL:        getEnv(envNotificationURL, "", true),
		useLabels:              getEnvBool(envUseLabels, false),
		notifyWhenHealthy:      getEnvBool(envNotifyWhenHealthy, true),
		checkStoppedContainers: getEnvBool(envCheckStoppedContainers, true),
		messagePrefix:          getEnv(envMessagePrefix, "", false),
		checkContainerExitCode: getEnvBool(envCheckContainerExitCode, false),
	}

	fmt.Println("Using config:")
	fmt.Println(fmt.Sprintf("  - failure limit: %v", c.failLimit))
	fmt.Println(fmt.Sprintf("  - cron schedule: %v", c.cronSchedule))
	fmt.Println(fmt.Sprintf("  - notification service: %v", strings.Split(c.notificationURL, "://")[0]))
	fmt.Println(fmt.Sprintf("  - use labels: %v", c.useLabels))
	fmt.Println(fmt.Sprintf("  - notify when healthy: %v", c.notifyWhenHealthy))
	fmt.Println(fmt.Sprintf("  - check stopped containers: %v", c.checkStoppedContainers))
	fmt.Println(fmt.Sprintf("  - message prefix: %v", c.messagePrefix))
	fmt.Println(fmt.Sprintf("  - check container exit code: %v", c.checkContainerExitCode))

	return c
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		ret, err := strconv.Atoi(value)
		if err == nil {
			return ret
		}
	}

	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		ret, err := strconv.ParseBool(value)
		if err == nil {
			return ret
		}
	}

	return fallback
}

func getEnv(key string, fallback string, trim bool) string {
	if value, ok := os.LookupEnv(key); ok {
		if trim {
			return strings.TrimSpace(value)
		} else {
			return value
		}
	}

	return fallback
}

func getDockerClient() *client.Client {
	cli, err := client.NewEnvClient()
	if err != nil {
		println("Error getting Docker Client, exiting...")
		panic(err)
	}

	return cli
}

func notify(notificationURL string, containerName string, healthy bool, messagePrefix string) {
	msg := fmt.Sprintf("%vContainer %v is not healthy", messagePrefix, containerName)
	if healthy {
		msg = fmt.Sprintf("%vContainer %v is back to healthy", messagePrefix, containerName)
	}

	currentTime := time.Now().Format(time.RFC3339)
	fmt.Println(fmt.Sprintf("%s | %s", currentTime, msg))

	senders := strings.Split(notificationURL, senderSplitter)
	for i := range senders {
		err := shoutrrr.Send(senders[i], msg)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error sending notification: %v", err))
		}
	}
}
