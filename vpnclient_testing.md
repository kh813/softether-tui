# SoftEther VPN Client Integration Testing & Verification Guide

This document records the exact procedure and results for setting up, running, and verifying `softether-vpnclient` integration with `softether-tui` in a Docker container environment.

---

## 1. Environment & Binaries

- **VPN Client Package**: `softether-vpnclient-v4.44-9807-rtm-2025.04.16-linux-arm64-64bit.tar.gz`
- **Download URL**: `https://github.com/SoftEtherVPN/SoftEtherVPN_Stable/releases/download/v4.44-9807-rtm/softether-vpnclient-v4.44-9807-rtm-2025.04.16-linux-arm64-64bit.tar.gz`
- **Target OS/Arch**: Linux arm64 (Docker container `softether-vpnserver`)
- **CLI Utility**: `/usr/vpnserver/vpncmd` or `/usr/vpnclient/vpncmd`

---

## 2. Setup & Installation Reproducibility Steps

### Step 1: Download & Extract Archive
```bash
curl -L -o softether-vpnclient.tar.gz https://github.com/SoftEtherVPN/SoftEtherVPN_Stable/releases/download/v4.44-9807-rtm/softether-vpnclient-v4.44-9807-rtm-2025.04.16-linux-arm64-64bit.tar.gz
docker cp softether-vpnclient.tar.gz softether-vpnserver:/tmp/softether-vpnclient.tar.gz
```

### Step 2: Build Daemon inside Container
```bash
docker exec softether-vpnserver apt-get update
docker exec softether-vpnserver apt-get install -y build-essential libc6-dev
docker exec softether-vpnserver bash -c "cd /tmp && tar -xzf softether-vpnclient.tar.gz && cd vpnclient && make main"
```

### Step 3: Install & Start `vpnclient` Service
```bash
docker exec softether-vpnserver bash -c "mkdir -p /usr/vpnclient && cp -r /tmp/vpnclient/* /usr/vpnclient/ && cd /usr/vpnclient && ./vpnclient start"
```

---

## 3. Verified Verification Test Cases

| ID | Test Category | Target Command / API | Command & Options | Expected Result | Verification Status |
|---|---|---|---|---|---|
| **CL-01** | Daemon Health | Process Status | `ps aux \| grep vpnclient` | `/usr/vpnclient/vpnclient execsvc` running | **PASS** |
| **CL-02** | Client Admin Auth | `AccountList` | `vpncmd /CLIENT localhost /CMD AccountList` | Status 0, connected to VPN Client `localhost` | **PASS** |
| **CL-03** | Account List Parsing | `AccountList` (CSV) | `vpncmd /CLIENT localhost /CSV /CMD AccountList` | Parsed columns (`VPN Connection Setting Name`, `Status`, `VPN Server Hostname`, `Virtual Hub`, `Virtual Network Adapter Name`) | **PASS** |
| **CL-04** | NIC List | `NicList` (CSV) | `vpncmd /CLIENT localhost /CSV /CMD NicList` | Parsed columns (`Virtual Network Adapter Name`, `Status`, `MAC Address`, `Version`) | **PASS** |
| **CL-05** | Account Creation | `AccountCreate` | `vpncmd /CLIENT localhost /CSV /CMD AccountCreate test /SERVER:127.0.0.1:443 /HUB:DEFAULT /USERNAME:User01 /NICNAME:VPN` | Exit code 0 | **PASS** |
| **CL-06** | Account Retrieval | `AccountGet` | `vpncmd /CLIENT localhost /CSV /CMD AccountGet test` | Parsed KeyValue map of account configuration | **PASS** |
| **CL-07** | Password Config | `AccountPasswordSet` | `vpncmd /CLIENT localhost /CSV /CMD AccountPasswordSet test /PASSWORD:Password123 /TYPE:standard` | Exit code 0 | **PASS** |
| **CL-08** | Account Deletion | `AccountDelete` | `vpncmd /CLIENT localhost /CSV /CMD AccountDelete test` | Exit code 0 | **PASS** |

---

## 4. Pending Test Cases for Next Review

The following items are candidate test cases for further expanding VPN Client integration coverage:

- [ ] **CL-09**: `AccountConnect` / `AccountDisconnect` verification (connecting to local VPN Hub).
- [ ] **CL-10**: `AccountDetailSet` (updating TCP connection counts, QoS, compression settings).
- [ ] **CL-11**: Virtual NIC creation (`NicCreate`) in environment with `TUN/TAP` device permissions (`/dev/net/tun`).
- [ ] **CL-12**: `AccountExport` / `AccountImport` configuration file operations.
