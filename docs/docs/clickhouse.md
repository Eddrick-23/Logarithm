# Usage of ClickHouse DB

Logarithm uses ClickHouse to persist logs. </br>
While log entries can be viewed using the React dashboard, the ClickHouse web UI is also accessible for manual database management.

By default, ClickHouse web UI is accessible at [localhost:8123](http://localhost:8123).
To access ClickHouse web UI services, the credentials specified in the docker-compose.yml must be used.</br>
By default these are:

```yaml
username: admin
password: strongpassword
```
