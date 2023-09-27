# NUXT Prometheus Exporter

NUXT Prometheus exporter makes it possible to monitor NUXT using Prometheus.


## Getting Started


1. download latest version from release
2. runas service

```bash
[Unit]
Description=NUXT Prometheus Exporter
After=network.target

[Service]
User=nuxt-exp
Group=nuxt-exp

ExecStart=/usr/local/bin/nuxt-prometheus-exporter
```

3. add server middleware to nuxt `server-middleware/prometheus.js`, example with axios

```js
import axios from 'axios';

export default function (req, res, next) {
  const start = +new Date();
  res.once('finish', () => {
    const end = +new Date();
    axios.post('http://localhost:45555/nodejs-requests', {
      route: req._parsedUrl.pathname ?? req.originalUrl,
      code: res.statusCode.toString(),
      method: req.method,
      date: start.toString(),
      duration: (end - start).toString(),
    }, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
  });
  next();
}
```

all values must be string

by analogy, can be used with any other project

4. enable server middleware in `nuxt.config.js`

```js
{
    // ...
    serverMiddleware: [
        // ...
        '~/server-middleware/prometheus',
        // ...
    ]
    // ...
}
```