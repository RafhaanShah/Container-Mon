
# Container-Mon

Get notified when your [Docker](https://www.docker.com/) containers are unhealthy.

## Prerequisites
- Have [Go](https://golang.org/) 1.17+ or [Docker](https://www.docker.com/) installed
- A notification service supported by [Shoutrrr](https://containrrr.dev/shoutrrr/services/overview/) and the required API keys or other configuration for your chosen service (e.g: Telegram, Discord, Slack, Teams etc)

## Configuration
All configuration is done via environment variables, see the table below for all options and default values. Only `CONTAINERMON_NOTIFICATION_URL` is mandatory, all other fields are optional.
| Name                            | Type   | Default Value         | Description                                                                                                                                         |
|---------------------------------|--------|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| CONTAINERMON\_FAIL\_LIMIT       | Int    | 1                     | Number of consecutive 'unhealthy' checks to reach before sending a notification                                                                     |
| CONTAINERMON\_CRON              | String | */5 * * * *           | Standard [Cron](https://crontab.guru/#*/5_*_*_*_*) schedule of when to run healthchecks                                                             |
| CONTAINERMON\_NOTIFICATION\_URL | String | N/A                   | Notification URL for [Shoutrrr](https://containrrr\.dev/shoutrrr/services/overview/)                                                                |
| CONTAINERMON\_USE\_LABELS       | Bool   | false                 | If `true` will only monitor containers with the label `containermon.enable=true` set                                                                |
| CONTAINERMON\_NOTIFY\_HEALTHY   | Bool   | true                  | If `true` will send a notification when an 'unhealthy' container returns to being 'healthy'                                                         |
| CONTAINERMON\_CHECK\_STOPPED    | Bool   | true                  | If `true` will consider `stopped` containers as 'unhealthy'\. If `false`, you will only be notified for containers that have a `healthcheck` set    |
| DOCKER\_HOST                    | String | /var/run/docker\.sock | Path for the Docker API socket                                                                                                                      |
| DOCKER\_API\_VERSION            | String | docker default        | Docker API version to use                                                                                                                           |
| DOCKER\_CERT\_PATH              | String | docker default        | Path to load the TLS certificates from                                                                                                              |
| DOCKER\_TLS\_VERIFY             | Bool   | false                 | Enable or disable TLS verification                                                                                                                  |

## Usage
- Stand-alone:
	`go run app.go`
- Docker:
  ```
	docker run \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-e CONTAINERMON_NOTIFICATION_URL=telegram://token@telegram?channels=channel-1 \
	ghcr.io/rafhaanshah/container-mon:latest
  ```
- Docker-Compose:
  ```
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

## Security Considerations
- It can be considered a security risk to directly map your Docket socket inside a container. A proxy such as [Socket-Proxy](https://github.com/Tecnativa/docker-socket-prox) can be used to give fine-grained access to parts of the Docker API, this application only needs to be able to read a list of running containers ->
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
