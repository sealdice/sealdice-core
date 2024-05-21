---
lang: zh-cn
title: QQ - Docker 中的海豹
---

# QQ - Docker 中的海豹

::: info 本节内容

本节将包含通过 docker 部署海豹时，你在 QQ 平台接入海豹核心需要了解的特定内容。

请至少完成 [QQ](./platform-qq) 一节中，[前言](./platform-qq/#前言)部分的阅读。

本节假定你对 `docker` 与 `docker compose` 有足够的了解。

:::

## 通过 `docker compose` 同时部署海豹与 Lagrange

通过此方式部署的海豹与 Lagrange 容器共同构成一个服务栈，可以方便地进行集中管理。请首先阅读 [QQ](./platform-qq) 一节中，[Lagrange](./platform-qq/#lagrange) 部分，大致了解 Lagrange 的部署过程。

### 创建 `docker-compose.yml`

首先，在工作目录下创建 `docker-compose.yml` 文件，并填入以下内容：

```yaml
name: sealdice

services:
  sealdice:
    image: ghcr.io/sealdice/sealdice:edge
    ports:
      - 3211:3211
    volumes:
      - ./seal_data:/data
      - ./seal_backups:/backups
    restart: unless-stopped

  lagrange:
    image: ghcr.io/konatadev/lagrange.onebot:edge
    volumes:
      - ./lagrange_data:/app/data
      - ./seal_data:/data
    restart: unless-stopped
```

此文件参考了[通过 docker 部署海豹](./quick-start/#启动)与[通过 docker 部署 Lagrange](https://github.com/LagrangeDev/Lagrange.Core/blob/master/Docker.md?tab=readme-ov-file) 相关内容。

此文件将宿主机 3211 端口映射到海豹容器的 3211 端口，如有需要，请根据实际情况自行调整端口映射。

此文件将工作目录下 `seal_data` 与 `seal_backups` 目录分别挂载到海豹容器的 `/data` 与 `/backups` 目录，并将 `lagrange_data` 与 `seal_data` 目录分别挂载到 Lagrange 容器的 `/app/data` 与 `/data` 目录。由于通过 QQ 后端发送本地图片时，海豹会将图片**在容器内**的绝对路径传递给 QQ 后端，所以需要将海豹数据也挂载到 Lagrange 容器以使 Lagrange 得以访问图片。如有需要，请根据实际情况自行调整挂载的目录。

::: warning 注意：在容器内以非 root 用户执行海豹进程可能会导致一些权限问题。

因此，示例文件以 root 用户生成容器进程。后续需要修改 `seal_data`、`seal_backups` 及 `lagrange_data` 目录中的内容（包括 Lagrange 配置文件、海豹数据等）时，需要 root 权限。

:::

::: details 补充：登录多个 QQ 账号

只需简单修改 `docker-compose.yml` 即可登录到多个 QQ 号：

```yaml
name: sealdice

services:
  sealdice:
    image: ghcr.io/sealdice/sealdice:edge
    user: <uid>:<gid>
    ports:
      - 3211:3211
    volumes:
      - ./seal_data:/data
      - ./seal_backups:/backups
    restart: unless-stopped

  lagrange-1:
    image: ghcr.io/konatadev/lagrange.onebot:edge
    environment:
      - UID=<uid>
      - GID=<gid>
    volumes:
      - ./lagrange_data_1:/app/data
    restart: unless-stopped

  lagrange-2:
    image: ghcr.io/konatadev/lagrange.onebot:edge
    environment:
      - UID=<uid>
      - GID=<gid>
    volumes:
      - ./lagrange_data_2:/app/data
    restart: unless-stopped
```

分别对每个 Lagrange 容器完成下述配置文件修改及扫码登录过程，并在下述海豹连接 Lagrange 填写 WS 地址时，将 `{Host}` 分别填入 `lagrange-1`、`lagrange-2` 即可。

:::

### 首次启动容器

在工作目录下使用以下命令启动容器：

```bash
docker compose up -d
```

首次启动容器后，`docker compose` 会自动创建 `seal_data`、`seal_backups` 以及 `lagrange_data` 目录。

### Lagrange 容器配置

首先使用以下命令停止容器运行：

```bash
docker compose stop
```

随后，按照 [QQ](./platform-qq) 一节中[运行 Lagrange](./platform-qq/#运行-lagrange) 部分修改 `lagrange_data/appsettings.json` 文件。需要特别注意的是，为了允许海豹容器正常访问 Lagrange 端口，需要将监听地址修改为 `0.0.0.0`：

`appsettings.json`：

```json{5}
{
  "Implementations": [
    {
      "Type": "ForwardWebSocket",
      "Host": "0.0.0.0",
      "Port": 8081,
      "HeartBeatInterval": 5000,
      "AccessToken": ""
    }
  ]
}
```

随后，通过 `docker compose up -d` 重新启动容器。通过 `docker compose logs lagrange` 访问 Lagrange 容器的日志，在日志中即可看到 QQ 登录二维码。同时 `lagrange_data/qr-0.png` 也是登录二维码。选择任一方式，尽快使用手机 QQ 扫码连接。

### 海豹连接 Lagrange

请参见 [QQ](./platform-qq) 一节中[海豹连接 Lagrange](./platform-qq/#海豹连接-lagrange) 部分。在填写 WS 正向服务地址 `ws://{Host}:{Port}` 时，`{Host}` 填写为 `lagrange` 即可，如果配置了多个 Lagrange 容器，填入对应服务的名称，`docker compose` 会自动处理主机名解析。`{Port}` 正常填写配置文件中设定的监听地址，在上文的例子中为 8081。

### 更新海豹容器或 Lagrange 容器

运行以下命令：

```bash
docker compose pull
docker compose up -d
```

当任一镜像有更新时，以上命令会完成容器更新。

## 连接到宿主机上的 QQ 后端

::: warning 注意：此种部署方式可能不能正常发送本地图片。

由于通过 QQ 后端发送本地图片时，海豹会将图片**在容器内**的绝对路径传递给 QQ 后端。宿主机上的 QQ 后端无法正确解析海豹数据目录的路径，因此可能无法正常发送图片。

:::

首先按照 [QQ](./platform-qq) 一节中的介绍，完成 QQ 后端的配置。

Docker 自动为容器构建了一个子网，容器默认接入此网络，并通过端口映射将容器端口暴露在宿主机。因此，容器中的海豹不能直接访问宿主机监听本地端口的 QQ 后端。需要连接时，有两种解决方案。

### 保持容器与宿主机网络隔离

Docker 为宿主机也分配了子网中的 IP，可以通过 `ip a` 等命令查看。例如：

```bash{5}
$ ip a
...
9: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN group default
    link/ether 02:42:33:50:ca:2d brd ff:ff:ff:ff:ff:ff
    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0
       valid_lft forever preferred_lft forever
    inet6 fe80::42:33ff:fe50:ca2d/64 scope link proto kernel_ll
       valid_lft forever preferred_lft forever
...
```

其中 `172.17.0.1` 为宿主机在 docker 子网中的 IP。

此时，首先修改 QQ 后端的监听设置，以确保其接受任何来源的连接（例如将 `127.0.0.1` 修改为 `0.0.0.0`），然后在配置海豹时，`{Host}` 填入宿主机在 docker 子网中的 IP（本例中为 `172.17.0.1`）。

### 配置容器使用宿主机网络

通过将 `docker run` 命令中的 `-p ...` 替换为 `--network host`，或在 `docker-compose.yml` 文件中将 `ports: ...` 替换为 `network_mode: host`，即可令容器使用宿主机网络。此时，在配置海豹时，`{Host}` 写为类似 `127.0.0.1` 或 `localhost` 即可正常访问监听本地端口的 QQ 后端。
