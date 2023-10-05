# Backname

Backname is a DNS server that gives every IP address its very own domain:

- **142.250.147.138.backname.io** resolves to **142.250.147.138**  
  _IPv4 with dots_
- **127-0-0-1.backname.io** resolves to **127.0.0.1**  
  _IPv4 with hyphens_
- **2a00.1450.401b.810.0.0.0.200e.backname.io** resolves to **2a00:1450:401b:810::200e**  
  _IPv6 with dots_
- **0--1.backname.io** resolves to **::1**  
  _IPv6 with hyphens_

The service is live publicly and for free over at [backname.io](https://backname.io), but feel free to host your own instance if you wish.

# Overview

Welcome to the Backname project, an open-source initiative that provides a unique domain for every IP address. This project is a Go application that sets up a DNS server, handling DNS requests and resolving them based on the environment variables set. The server listens on port 53 and is built using a Dockerfile with a base image of golang:1.21-alpine. The built application is then copied into a distroless static-debian12 image. The project also includes a webpage layout with information about Backname™, a service that provides a backname for every IP address.

# Technologies and Frameworks

The Backname project utilizes a variety of technologies and frameworks:

- **Go**: The main programming language used in the project.
- **Docker**: Used for building and running the application.
- **Ruby**: Used in the Gemfile and Gemfile.lock for managing Ruby gems.
- **Jekyll**: Used for the website, with the jekyll-seo-tag plugin.
- **CSS**: Used for styling the webpage.

# Installation

## Prerequisites

Before you begin, ensure you have met the following requirements:

- You have installed the latest version of Go.
- You have installed Docker.
- You have installed Ruby and Bundler.

## Installing the Project

To install the project, follow these steps:

1. Clone the repository:

```bash
git clone https://github.com/your-username/your-project.git
```

2. Navigate to the project directory:

```bash
cd your-project
```

3. Install the required Go modules:

```bash
go get github.com/Twixes/backname
go get github.com/miekg/dns@v1.1.56
go get golang.org/x/mod@v0.12.0
go get golang.org/x/net@v0.15.0
go get golang.org/x/sys@v0.12.0
go get golang.org/x/tools@v0.13.0
```

4. Build the Docker image:

```bash
docker build -t your-project .
```

5. Run the Docker container:

```bash
docker run -p 53:53 your-project
```

6. Install the required Ruby gems:

```bash
bundle install
```

7. Start the Jekyll server:

```bash
bundle exec jekyll serve
```

## Using the Project

After installation, you can use the project by navigating to `http://localhost:4000` in your web browser.
