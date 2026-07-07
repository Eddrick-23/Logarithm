# Usage of Clickhouse DB

Logarithm uses Clickhouse to persist logs. </br>
While log entries can be viewed using the react dashboard, the clickhouse web ui is also accessible for manual database management.

By default, clickhouse web ui is accessible at [localhost:8123](http://localhost:8123).
To access clickhouse web ui services, the credentials specified in the dockercompose.yaml must be used.</br>
By default these are:
```
username: admin
password: strongpassword
```
