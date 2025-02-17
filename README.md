# Platypus

**Dynamic Platform for Deployment of Microservices Based on Energy Efficiency**

Platypus is an innovative platform designed to optimize the deployment of microservices with a focus on minimizing energy consumption and reducing carbon footprint. By analyzing workload metrics and energy efficiency data from both cloud providers and on-premise data centers, Platypus makes intelligent decisions about microservice placement to maximize performance while conserving energy.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Technical Implementation](#technical-implementation)
- [Environment Setup](#environment-setup)
  - [AWS](#aws)
  - [GCP](#gcp)
  - [Azure](#azure)
- [Installation & Running](#installation--running)
- [Configuration](#configuration)
- [TODO](#todo)
- [License](#license)

## Overview

Platypus is engineered to support modern microservice architectures by providing the following key benefits:

- **Energy Efficiency Analysis:** Monitor server and container energy consumption metrics from various cloud providers and local data centers. Analyze historical and real-time data to determine the optimal strategies for energy consumption minimization.
- **Intelligent Migration Planning:** Leverage machine learning algorithms to determine the best moments and servers for migrating microservices. Enable dynamic container migrations using container orchestration tools such as Kubernetes, reducing the load on power-intensive servers.
- **EcoTags:** Classify services using eco-tags based on their energy usage and carbon footprint. Visualize service impact through an integrated dashboard.
- **Dynamic Autoscaling:** Automatically adjust the number of instances based on current workload demands and energy consumption metrics, ensuring efficient resource utilization.
- **Eco-network Management:** Visualize and manage energy consumption in real-time through an interactive dashboard. Integrate with APIs for seamless energy management and green computing.

## Features

- **Energy Efficiency Monitoring:** Collect and analyze energy consumption data.
- **Machine Learning Integration:** Use ML algorithms to plan optimal microservice migrations.
- **Service Classification:** Implement eco-tags for energy-based classification.
- **Real-time Visualization:** Dashboard for monitoring energy and performance metrics.
- **Dynamic Scaling:** Auto-adjust microservice instances based on demand.
  
## Technical Implementation

Platypus is developed using the Go programming language due to its high performance and low resource consumption. The platform is containerized using Docker and orchestrated with Kubernetes. Integration with cloud providers (AWS, GCP, and Azure) is built into the system to ensure seamless data collection and operational control. Where applicable, internal modules and libraries have been used in alignment with organizational best practices.

## Environment Setup

Before running the project, set up the required environment variables as per your selected cloud provider.

### AWS

Set the following environment variables for AWS access:

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=your_region
```

GCP
Ensure that you set the environment variable for GCP credentials:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

Azure
Set the necessary credentials for Azure:
```bash
export AZURE_TENANT_ID=your_tenant_id
export AZURE_CLIENT_ID=your_client_id
export AZURE_CLIENT_SECRET=your_client_secret
```

Installation & Running
To quickly launch the project, run the following command in your terminal:

```bash
go run ./cmd/server/main.go
```

To build an executable binary, use:

```bash
go build -o platypus ./cmd/server/main.go
```

After building, run the binary:

```bash
./platypus
```

The server will start and listen on port 8080. Check the console logs for a message indicating the successful launch of the Platypus server.


Configuration
Additional configurations may be required for:


Connecting to cloud provider APIs.
Integrating with Kubernetes for container orchestration.
Customizing modules and internal settings within configuration files and accompanying modules.
Please update these settings based on your deployment environment and requirements.


TODO
Integration with Kubernetes for enhanced container management and orchestration.
Integration with Azure Cloud for Azure-specific features and services.


License
Apache 2.0
