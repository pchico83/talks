## Backend

Deploy your dev mode environment by executing:

```
cd backend
okteto up $USERNAME/api
```

Execute commands in you dev environment by executing:

```
okteto exec command
```

To hot-reload your go binaries for every code change: 
```
okteto exec -- go get github.com/codegangsta/gin
okteto exec -- gin -p 8000 run
```

To run the tests:
```
 okteto exec go test ./...
 ```

### Local dependency management
* `backend/Gopkg.toml` contains the versions of everything.
* To add a single dependency: `dep ensure -add github.com/pkg/errors`
* To update a single dependency: `dep ensure -update github.com/foo/bar`
* To update all dependencies: `dep ensure -update`
* To sync your local development: `dep ensure`

More [on dep information here](https://golang.github.io/dep/docs/daily-dep.html)

### Configuration
The application will look for a configuration file in the following locations:
- $APP_DIR/config/config.yml
- $APP_DIR/config.yml

The application will panic if it can't find a configuration file.

Any configuration attribute can be overriden by an environment variable. The
variable must be all in upper case, have `OKTETO_` as a prefix and use underscore ( `_` ) as the separator:
- OKTETO_DATABASE_HOST=myhost
- OKTETO_DATABASE_PORT=5460

This must be set before starting the application

### Data migrations
To add a data migration:
- Create the following files (version is the highest existing version + 1)
-- {version}_{name}.up.sql
-- {version}_{name}.down.sql
- Update `currentSchema` in `store/sql_store.go`to match the version number you selected

Upon start, the data migration will automatically run before starting the APIs