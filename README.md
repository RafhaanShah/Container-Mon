
# Container-Mon

Get notified when your [Docker](https://www.docker.com/) containers are unhealthy.

![](/assets/screenshot.jpg)

## Prerequisites
- A notification service supported by [Shoutrrr](https://containrrr.dev/shoutrrr/services/overview/) and the required API keys or other configuration for your chosen service (e.g: Telegram, Discord, Slack, Teams etc)

## Building
Build the app:
```
go build
```
Format:
```
gofmt -w -s .
```

## Configuration
Configuration can be set via **environment variables** or **command line flags**. Command line flags take precedence over environment variables. Only `CONTAINERMON_NOTIFICATION_URL` (or `--notification-url`) is mandatory; all other fields are optional.

| Environment Variable                | Command Line Flag                | Type   | Default Value         | Description                                                                                                                                                                  |
|-------------------------------------|----------------------------------|--------|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| CONTAINERMON_FAIL_LIMIT             | --fail-limit                     | Int    | 1                     | Number of consecutive 'unhealthy' checks to reach before sending a notification                                                                                              |
| CONTAINERMON_CRON                   | --cron                           | String | */5 * * * *           | Standard [Cron](https://crontab.guru/#*/5_*_*_*_*) schedule of when to run healthchecks                                                                                      |
| CONTAINERMON_NOTIFICATION_URL       | --notification-url               | String | N/A                   | Notification URL for [Shoutrrr](https://containrrr.dev/shoutrrr/services/overview/). Multiple services can be used with the `|` (pipe) character as a separator.            |
| CONTAINERMON_HEALTHY_NOTIFICATION_URL| --healthy-notification-url      | String | N/A                   | Notification URL for healthy state notifications                                                                                                                             |
| CONTAINERMON_USE_LABELS             | --use-labels                     | Bool   | false                 | If `true`, only monitor containers with the label `containermon.enable=true` set                                                                                             |
| CONTAINERMON_NOTIFY_HEALTHY         | --notify-healthy                 | Bool   | true                  | If `true`, send a notification when an 'unhealthy' container returns to being 'healthy'                                                                                      |
| CONTAINERMON_CHECK_STOPPED          | --check-stopped                  | Bool   | true                  | If `true`, consider `stopped` containers as 'unhealthy'. If `false`, only containers with a `healthcheck` set are monitored                                                  |
| CONTAINERMON_MESSAGE_PREFIX         | --message-prefix                 | String | N/A                   | Custom text to be prefixed to all notification messages.                                                                                                                     |
| CONTAINERMON_CHECK_EXIT_CODE        | --check-exit-code                | Bool   | false                 | When set to `true`, only exited containers with a non-zero exit code will be marked as unhealthy                                                                             |
| DOCKER_HOST                         |                                  | String | /var/run/docker.sock  | Path for the Docker API socket                                                                                                                                                |
| DOCKER_API_VERSION                  |                                  | String | docker default        | Docker API version to use                                                                                                                                                    |
| DOCKER_CERT_PATH                    |                                  | String | docker default        | Path to load the TLS certificates from                                                                                                                                       |
| DOCKER_TLS_VERIFY                   |                                  | Bool   | false                 | Enable or disable TLS verification                                                                                                                                            |

## Usage
### Stand-alone

```shell
go run app.go --notification-url "telegram://token@telegram?channels=channel-1" --fail-limit=3 --cron="*/2 * * * *"
```

### Docker

```shell
docker run \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e CONTAINERMON_NOTIFICATION_URL=telegram://token@telegram?channels=channel-1 \
  ghcr.io/rafhaanshah/container-mon:latest
```

### Docker-Compose

```yaml
version: "3.8"
services:
  container-mon:
    container_name: container-mon
    image: ghcr.io/rafhaanshah/container-mon:latest
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - CONTAINERMON_NOTIFICATION_URL=telegram://token@telegram?channels=channel-1
```

## Troubleshooting
- Docker API version issues: if you get error messages like `client version 1.43 is too new. Maximum supported API version is 1.42` then please set the `DOCKER_API_VERSION` environment variable to the latest version supported by your Docker engine (e.g. `DOCKER_API_VERSION=1.42`, which you can check by running `docker version`.
- Notifier issues: please check if your URL works with the Shoutrrr CLI from [here](https://containrrr.dev/shoutrrr/0.7/getting-started/#through_the_cli).

## Security Considerations
- It can be considered a security risk to directly map your Docket socket inside a container. A proxy such as [Socket-Proxy](https://github.com/Tecnativa/docker-socket-proxy) can be used to give fine-grained access to parts of the Docker API, this application only needs to be able to read a list of running containers ->
  ```
	docker run \
	-e DOCKER_HOST=tcp://socket-proxy:2375
	...
   ```
- This container runs as `root` by default to access the Docker socket. You may run it as another user that has access to the socket as described here: [Running a Docker container as a non-root user](https://medium.com/redbubble/running-a-docker-container-as-a-non-root-user-7d2e00f8ee15) ->
  ```
	docker run \
	-u $(id -u):$(stat -c '%g' "/var/run/docker.sock") \
	...
   ```

## License
[MIT](https://choosealicense.com/licenses/mit/)
