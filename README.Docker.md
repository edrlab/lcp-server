### Setting secrets
A http basic authentication username and password, which are allowing access to the server API private endpoints, are managed via docker secrets.

Please update /config/basic-auth.txt with the data you want to use for accessing the LCP Server:
- the first line is the username (replace "username" by a secure value),
- the second line is the password (replace "password" by a secure value).

### Building and running your application

Build the image of an LCP Server using SQLite and start the containers:
`docker compose up -d`
or
`docker compose up`
if you prefer seeing application logs in the terminal 

The image is named `lcp-server`. 

Note: to force the image to be rebuilt, type:
`docker compose up --build`

Your application will be available at http://localhost:8989.

### Alternative builds
Build the image of an LCP server using SQLite by running:
`docker compose build --tag lcp-server:sqlite .`

Build the image of an LCP server using PostgresQL by running:

### Run in detached mode

If you choose to expose the default port, use:
`docker run --detach --publish 8989:8989 lcp-server`

### Deploying your application to the cloud

First, build your image, e.g.: `docker build -t lcp-server .`.
If your cloud uses a different CPU architecture than your development
machine (e.g., you are on a Mac M1 and your cloud provider is amd64),
you'll want to build the image for that platform, e.g.:
`docker build --platform=linux/amd64 -t lcp-server .`.

Then, push it to your registry, e.g. `docker push myregistry.com/lcp-server`.

Consult Docker's [getting started](https://docs.docker.com/go/get-started-sharing/)
docs for more detail on building and pushing.

### References
* [Docker's Go guide](https://docs.docker.com/language/golang/)