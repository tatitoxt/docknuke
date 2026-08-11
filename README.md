# ☢️ DockNuke

> High-Performance Go CLI utility for smart cleaning, filtering, and disk optimization of Docker environments.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Engine_API-2496ED?style=flat&logo=docker&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

---

## 🌟 Key Features

- ⚡ **Smart Pruning**: Instantly scan and reclaim gigabytes of disk space from dangling containers, dead images, build caches, and unattached volumes.
- 🛡️ **Dry-Run Safety Engine**: `--dry-run` flag previews exactly what will be deleted and how much disk space will be reclaimed before touching any resources.
- ⏳ **Retention Filters**: Keep recent resources with `--keep-days N`.
- 📊 **Disk Reclaim Telemetry**: High-precision space breakdown reported in human-readable formats.

---

## 🚀 Quick Start

### Installation

```bash
git clone https://github.com/tatitoxt/docknuke.git
cd docknuke
go build -o docknuke ./cmd/docknuke
```

### Usage

```bash
# Preview reclaimable disk space (Dry Run)
./docknuke --dry-run

# Reclaim space keeping resources created in the last 7 days
./docknuke --keep-days 7
```

---

## 👤 Author

Developed by **Fausto Pastura** ([@tatitoxt](https://github.com/tatitoxt))  
Salesforce Consultant & Forward Deployed AI Engineer.
