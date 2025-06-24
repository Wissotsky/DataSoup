# DataSoup

A feed of changes from data.gov.il (Open Goverment Data)

Currently publishes to telegram channel: [DataSoup](https://t.me/DataSoup)

## How to run

### Local Development

Create a file name `.telegram_token` with the telegram bot token.

Find and replace the `chat_id` in `main.go` with the chat id of the channel you want to publish to.

Build the project by running `go build`

**Bootstrap the data (REQUIRED for first run):**
```bash
./main -bootstrap
```
This goes back one week and takes up about 5.5 GB of disk space.

Run the project by running `main`

### Docker Deployment

Build the Docker image:
```bash
docker build -t datasoup .
```

**Bootstrap the data (REQUIRED for first run):**
```bash
docker run -e TELEGRAM_TOKEN=your_token_here -v ./data:/root/data datasoup ./main -bootstrap
```

Run the monitoring server:
```bash
docker run -p 8080:8080 -e TELEGRAM_TOKEN=your_token_here -v ./data:/root/data datasoup
```

Run the data collection worker:
```bash
docker run -e TELEGRAM_TOKEN=your_token_here -v ./data:/root/data datasoup ./main
```

### Using Docker Compose

Set your telegram token in environment variables:
```bash
export TELEGRAM_TOKEN=your_token_here
```

**Bootstrap data (REQUIRED for first run):**
```bash
docker-compose run datasoup-worker ./main -bootstrap
```

Run the monitoring server:
```bash
docker-compose up datasoup-monitoring
```

Run the worker (for data collection):
```bash
docker-compose run datasoup-worker
```

### Deployment with Coolify

1. Set `TELEGRAM_TOKEN` environment variable in Coolify
2. Deploy using the provided docker-compose.yml
3. **IMPORTANT: Bootstrap data before first run:**
   - Run a one-time job: `./main -bootstrap`
   - Or use Coolify's command runner to execute the bootstrap
4. The monitoring server will be available on port 8080
5. Set up a cron job to run the worker 2-3 times per day

**Bootstrap Command for Coolify:**
```bash
./main -bootstrap
```

Every time you run the project it will go through[1] any changes made since last time, diff them and publish them to the telegram channel.

1. Currently its artificially **significantly** slowed down and runs sequentially to ease debuggging and avoid telegram api rate limits.

## Important Notes

- **You MUST run bootstrap before the first normal run**, otherwise the script will fail because there's no `data/packagedata.json` file to compare against
- Bootstrap downloads data from the last week (~5.5GB) and creates the initial state
- After bootstrap, regular runs will only process changes since the last run
- The bootstrap process may take 30-60 minutes depending on your connection

## Monitoring Server

A simple HTTP monitoring server is available to view the current status of DataSoup:

Build the monitoring server: `go build -o monitoring_server monitoring_server.go`

Run the monitoring server: `./monitoring_server`

The server will start on port 8080 and display:
- Last update timestamp from the packagedata.json file
- Table of all datasets sorted by last modified date
- Clickable links to view datasets on data.gov.il
- Dataset metadata including organization, resource count, and tags

## What are Resources Datasets and Organizations?

**Organizations** are the entities that publish the data. E.g. Ministry of Health, Ministry of Education, etc.

**Datasets** is the logical grouping of data. E.g. Covid-19 data, School data, etc.

**Resources** consist of the actual data files. E.g. CSV files, XLSX files, etc. (We only care about CSV files for now)

**Tags** group Datasets by a common theme. E.g. Covid-19, Education, etc.
