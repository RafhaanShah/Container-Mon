package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containrrr/shoutrrr"
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
	envHealthyNotificationURL = "CONTAINERMON_HEALTHY_NOTIFICATION_URL"
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
	healthyNotificationURL string
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
			fmt.Printf("Error checking containers: %v\n", err)
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
				notify(conf, c.Names[0], true)
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
					notify(conf, c.Names[0], false)
				}
			}
		}
	}

	return nil
}

func containerInList(a string, list []container.Summary) bool {
	for _, b := range list {
		if b.ID == a {
			return true
		}
	}

	return false
}

func getContainers(ctx context.Context, cli *client.Client, filterByLabel bool, checkStoppedContainers bool) ([]container.Summary, error) {
	args := filters.NewArgs()
	if filterByLabel {
		args.Add("label", fmt.Sprintf("%v=true", containerLabel))
	}

	return cli.ContainerList(ctx, container.ListOptions{
		All:     checkStoppedContainers,
		Filters: args,
	})
}

func isHealthy(ctx context.Context, cli *client.Client, c container.Summary, conf config) bool {
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
	}

	if err != nil {
		return running
	}

	health := containerJSON.State.Health
	if health == nil || health.Status == container.NoHealthcheck || health.Status == container.Starting {
		return running
	}

	return health.Status == container.Healthy
}

func getConfig() config {
	failLimitFlag := flag.Int("fail-limit", getEnvInt(envFailLimit, 1), "Number of consecutive 'unhealthy' checks before notification")
	cronScheduleFlag := flag.String("cron", getEnv(envCronSchedule, "*/5 * * * *", true), "Cron schedule for healthchecks")
	notificationURLFlag := flag.String("notification-url", getEnv(envNotificationURL, "", true), "Notification URL for Shoutrrr")
	healthyNotificationURLFlag := flag.String("healthy-notification-url", getEnv(envHealthyNotificationURL, "", true), "Notification URL for healthy state")
	useLabelsFlag := flag.Bool("use-labels", getEnvBool(envUseLabels, false), "Monitor only containers with containermon.enable=true label")
	notifyWhenHealthyFlag := flag.Bool("notify-healthy", getEnvBool(envNotifyWhenHealthy, true), "Notify when unhealthy container returns to healthy")
	checkStoppedContainersFlag := flag.Bool("check-stopped", getEnvBool(envCheckStoppedContainers, true), "Consider stopped containers as unhealthy")
	messagePrefixFlag := flag.String("message-prefix", getEnv(envMessagePrefix, "", false), "Custom prefix for notification messages")
	checkContainerExitCodeFlag := flag.Bool("check-exit-code", getEnvBool(envCheckContainerExitCode, false), "Only exited containers with non-zero exit code are unhealthy")

	flag.Parse()

	c := config{
		failLimit:              *failLimitFlag,
		cronSchedule:           *cronScheduleFlag,
		notificationURL:        *notificationURLFlag,
		healthyNotificationURL: *healthyNotificationURLFlag,
		useLabels:              *useLabelsFlag,
		notifyWhenHealthy:      *notifyWhenHealthyFlag,
		checkStoppedContainers: *checkStoppedContainersFlag,
		messagePrefix:          *messagePrefixFlag,
		checkContainerExitCode: *checkContainerExitCodeFlag,
	}

	// Fallback logic for healthy notification URL
	if c.healthyNotificationURL == "" {
		c.healthyNotificationURL = c.notificationURL
	}

	fmt.Println("Using config:")
	fmt.Printf("  - failure limit: %v\n", c.failLimit)
	fmt.Printf("  - cron schedule: %v\n", c.cronSchedule)
	fmt.Printf("  - notification service: %v\n", strings.Split(c.notificationURL, "://")[0])
	fmt.Printf("  - healthy notification service: %v\n", strings.Split(c.healthyNotificationURL, "://")[0])
	fmt.Printf("  - use labels: %v\n", c.useLabels)
	fmt.Printf("  - notify when healthy: %v\n", c.notifyWhenHealthy)
	fmt.Printf("  - check stopped containers: %v\n", c.checkStoppedContainers)
	fmt.Printf("  - message prefix: %v\n", c.messagePrefix)
	fmt.Printf("  - check container exit code: %v\n", c.checkContainerExitCode)

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
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Error getting Docker Client, exiting...")
		panic(err)
	}

	return cli
}

func notify(conf config, containerName string, healthy bool) {
	msg := fmt.Sprintf("%vContainer %v is not healthy", conf.messagePrefix, containerName)
	url := conf.notificationURL
	if healthy {
		msg = fmt.Sprintf("%vContainer %v is back to healthy", conf.messagePrefix, containerName)
		url = conf.healthyNotificationURL
	}

	currentTime := time.Now().Format(time.RFC3339)
	fmt.Printf("%s | %s\n", currentTime, msg)

	senders := strings.Split(url, senderSplitter)
	for i := range senders {
		err := shoutrrr.Send(senders[i], msg)
		if err != nil {
			fmt.Printf("Error sending notification: %v\n", err)
		}
	}
}
