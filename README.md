# Backname

Backname is a DNS server that gives every IP address its very own domain:

- **142.250.147.138.backname.io** resolves to **142.250.147.138**  
  _IPv4 with dots_
- **127-0-0-1.backname.io** resolves to **127.0.0.1**  
  _IPv4 with dashes_
- **2a00.1450.401b.810.0.0.0.200e.backname.io** resolves to **2a00:1450:401b:810::200e**  
  _IPv6 with dots_
- **0--1.backname.io** resolves to **::1**  
  _IPv6 with dashes_

The service is live publicly and for free over at [backname.io](https://backname.io), but feel free to host your own instance if you wish.

## Self-hosting

Hosting Backname yourself is easy with `docker compose`.

The only prerequisite: a Linux server instance with a public IPv4 address attached, and with `git` + `docker compose` installed.

With the server instance ready, follow the steps below:

1. Get Backname onto your disk, e.g. in your home directory (`~`):

    ```bash
    git clone https://github.com/Twixes/backname.git
    ```

2. Enter the `backname` directory:

    ```bash
    cd backname
    ```

3. Use your favorite text editor to create the `.env` file storing configuration:

    ```bash
    nano .env
    ```

    This file must follow the format below. Only `ZONE` and `NAMESERVER_A` are _required_ for operation:

    ```bash
    # The domain name under which Backname will be running (real-world example: backname.io)
    ZONE=your-backname-domain.com
    # The public IPv4 address of this server hosting Backname (in a dual-server setup, two comma-separated addresses)
    NAMESERVER_A=123.123.123.123
    # Optional: The public IPv6 address of this server hosting Backname, if supporting IPv6 (in a dual-server setup, two comma-separated addresses)
    NAMESERVER_AAAA=
    # Optional: Website A and/or AAAA records that will be served for your-backname-domain.com + www.your-backname-domain.com (comma-separated)
    WEBSITE_A=
    WEBSITE_AAAA=
    # Optional: TXT record values server at the root of the zone (comma-separated), if needed for e.g. domain verification
    ROOT_TXT=
    # Optional: IP addresses blocked from receiving a backname (comma-separated), if seeing problematic usage
    BLOCKLIST=
    ```

    Once done, save the `.env` file.

4. Start the Backname server:

    ```bash
    docker compose up -d
    ```

    > In older versions of Docker, the command may be `docker-compose` with a hyphen.

5. Verify that startup succeeded:

    ```bash
    docker compose logs
    ```

    You should be seeing `DNS server listening on :53` at the very top. If that is the case, the Backname server is now ready to process DNS queries!

6. The final step is to configure your domain (`ZONE`) to use this server for its own DNS resolution:

    1. Go to your domain registrar's DNS settings for the domain.
    2. Set the domain's nameservers to:

          ```plaintext
          alpha.your-backname-domain.com
          ```

        <details>
          <summary>Dual-server setup</summary>

          ```plaintext
          alpha.your-backname-domain.com
          omega.your-backname-domain.com
          ```

        </details>

          Do not change try to change these subdomains from `alpha` and `omega` to something else – they are hard-coded.

    3. Also set **glue records** so that the nameservers can be found initially:

          ```plaintext
          alpha.your-backname-domain.com → value of NAMESERVER_A
          ```

          If supporting IPv6:

          ```plaintext
          alpha.your-backname-domain.com → value of NAMESERVER_A
          alpha.your-backname-domain.com → value of NAMESERVER_AAAA
          ```

        <details>
          <summary>Dual-server setup</summary>

       ##### IPv4-only

        ```plaintext
        alpha.your-backname-domain.com → first value in NAMESERVER_A
        omega.your-backname-domain.com → second value in NAMESERVER_A
        ```

       ##### IPv4 + IPv6

        ```plaintext
        alpha.your-backname-domain.com → first value in NAMESERVER_A
        alpha.your-backname-domain.com → first value in NAMESERVER_AAAA
        omega.your-backname-domain.com → second value in NAMESERVER_A
        omega.your-backname-domain.com → second value in NAMESERVER_AAAA
        ```

        </details>

7. Now wait for DNS propagation to complete. It can be quick but _may_ take up to 48 hours.

    Use `dig NS your-backname-domain.com` to check the status of propagation. Once this command returns `alpha.your-backname-domain.co` – your Backname DNS is operational!

### Achieving high availability

For redundancy, you should host two Backname instances in different data centers. In that case everything stays the same, except that `NAMESERVER_A` (and optionally `NAMESERVER_AAAA` too) contains two comma-separated IP address values, rather than just one (refer to "dual-server setup" annotations in the steps above).
