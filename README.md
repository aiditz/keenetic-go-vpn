## Description 

`keenetic-go-vpn` is an alternative web control panel for Keenetic routers, focused on fine‑grained control of internet access and VPN routing.

It talks directly to the router over the Keenetic HTTP RCI API and gives you:

#### 1. Device management UI
- Per‑device internet access control (permit/deny)
- Per‑device IP policy selection (e.g. “NoVPN”), with persistent config save
#### 2. Domain‑based VPN routing
- Dedicated page to manage a list of domains and route their traffic via a chosen router interface (e.g. Wireguard0).
- For each domain:
- - Automatic nslookup to resolve current IPv4 addresses.
- - Manual IP override/edit.
- - One‑click “Apply” to create/update routes on the router.
- Routes are created via ip/route with aggregated comments like: "[GOVPN] example.com, other.example.com".
- “Sync All” button to re‑resolve all domains and update routes on the router.
#### 3. Background sync & persistence
- Optional Auto refresh all routes: when enabled, the backend runs once per day at 00:00 UTC, resolves all domains and updates the routes.
- All domain routing data (domains, IPs, active flags, selected interface, auto‑refresh setting) is stored in a JSON file on a Docker volume, so configuration survives container restarts.
#### 4. No session expiration every 10 minutes
- This is an annoying limitation in the original web panel. With `keenetic-go-vpn` you can control the session lifetime via `.env` and keep the browser tab open as long as you want.

### Stack
Backend: Go, Gin, Keenetic RCI client (HTTP + batch RCI calls). ~7 Mb RAM.

Frontend: Vue 3 + Tailwind CSS, single‑page UI.

## Installation with Docker

Clone the repo:
```bash
git clone https://github.com/vmuromskii/keenetic-go-vpn.git
cd keenetic-go-vpn
cp .env.example .env
```
Configure your credentials:
```bash
nano .env
```

Run the container:
```bash
docker-compose up -d --build
```

Open http://localhost:800.

You’re all set!
