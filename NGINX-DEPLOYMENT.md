# LCP Server Deployment with Nginx

This guide explains how to deploy the LCP server with nginx as a reverse proxy.

## Architecture

```
Internet → Nginx (port 80/443) → LCP Server (port 8989) → MySQL (port 3306)
```

## Configuration

### Docker Compose Services

1. **nginx**: Reverse proxy and entry point
   - Ports: 80 (HTTP) and 443 (HTTPS ready)
   - Configuration: `config/nginx.conf`
   - Rate limiting and security configured

2. **server**: LCP Server application
   - Internal port: 8989 (not directly exposed)
   - Health check configured

3. **mysql**: MySQL 8.4.7 database
   - Port: 3307 (mapped from 3306)

### Nginx Features

- **Rate limiting**: 10 requests/second per IP
- **Gzip compression** for performance optimization
- **Security headers** (XSS, CSRF protection, etc.)
- **Automatic health checks**
- **Access and error logs**
- **Upstream with keep-alive** for better performance

### Configured Endpoints

- `/health`: Health check (no rate limiting)
- `/dashboard`: Dashboard with authentication
- `/api/*`: API endpoints with moderate rate limiting
- `/license/*` and `/lsd/*`: LCP endpoints with high rate limiting
- `/*`: All other endpoints

## Deployment

### 1. Starting Services

```bash
# Build and start services
docker compose up -d

# Check logs
docker compose logs -f nginx
docker compose logs -f server
```

### 2. Verification

```bash
# Test health check
curl http://localhost/health

# Test dashboard (if configured)
curl http://localhost/dashboard
```

### 3. Monitoring

```bash
# Check services status
docker compose ps

# View nginx logs
docker compose logs nginx

# Check health metrics
docker compose exec nginx nginx -t
```

## HTTPS Configuration (Optional)

To enable HTTPS, uncomment the SSL section in `config/nginx.conf` and:

1. Place your certificates in `config/ssl/`
2. Add the SSL volume in `compose.yaml`:
   ```yaml
   volumes:
     - ./config/ssl:/etc/ssl:ro
   ```
3. Configure appropriate domain names

### Example with Let's Encrypt

```bash
# Create SSL directory
mkdir -p config/ssl

# Obtain certificates (example with certbot)
certbot certonly --webroot -w /var/www/html -d your-domain.com
cp /etc/letsencrypt/live/your-domain.com/* config/ssl/
```

## Customization

### Rate Limiting

Modify in `config/nginx.conf`:
```nginx
# Adjust request rate
limit_req_zone $binary_remote_addr zone=api:10m rate=20r/s;
```

### Custom Logs

```nginx
# Add specific logs
access_log /var/log/nginx/lcp-access.log main;
```

### Additional Security Headers

```nginx
# CSP for enhanced security
add_header Content-Security-Policy "default-src 'self';" always;
```

## Troubleshooting

### Common Issues

1. **502 Bad Gateway**: LCP service is not accessible
   ```bash
   docker compose logs server
   docker compose restart server
   ```

2. **Timeout errors**: Adjust nginx timeouts
   ```nginx
   proxy_connect_timeout 60s;
   proxy_read_timeout 180s;
   ```

3. **Rate limiting too restrictive**:
   ```nginx
   limit_req zone=api burst=50 nodelay;
   ```

### Diagnostic Commands

```bash
# Test nginx configuration
docker compose exec nginx nginx -t

# Reload configuration without restart
docker compose exec nginx nginx -s reload

# Connection stats
docker compose exec nginx nginx -s status
```

## Production

### Production Recommendations

1. **Enable HTTPS** with valid certificates
2. **Configure automatic backups** for the database
3. **Monitor logs** with systems like ELK or Grafana
4. **Adjust limits** according to expected load
5. **Configure OS-level firewall**
6. **Use Docker secrets** for passwords (already configured)

### Production Environment Variables

```bash
# .env file for production
NGINX_WORKER_PROCESSES=auto
NGINX_WORKER_CONNECTIONS=2048
MYSQL_MAX_CONNECTIONS=200
```