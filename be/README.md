# Doing Now - 后端服务 Docker 部署指南

本文档介绍如何从零开始，使用 Docker 在本地编译、部署项目，并初始化数据库环境。

## 1. 前置要求

确保本地已安装：
*   [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Windows/Mac) 或 Docker Engine (Linux)
*   确保 Docker 正在运行。

## 2. 目录结构说明

```text
be/
├── conf/
│   └── deploy.local.yml   # 应用配置文件（已适配 Docker 网络）
├── docs/
│   └── sql/
│       └── init.sql       # 数据库建表 SQL
├── Dockerfile             # 多阶段构建文件
├── docker-compose.yaml    # 容器编排配置
└── main.go                # 入口文件
```

## 3. swag初始化

在项目根目录 (`be/`) 下执行以下命令初始化 Swagger 文档：

```bash
swag init -g main.go -o docs

swag fmt
```

## 4. 单独构建与运行 (可选)

如果你不使用 docker-compose，仅想手动管理应用镜像和容器，可以使用以下命令。

### 1. 构建镜像 (Build)
在项目根目录 (`be/`) 下执行：

```bash
docker build -t be-app:latest .
```

### 2. 运行容器 (Run)
启动容器并映射端口 (8000)：

```bash
docker run -d -p 8000:8000 --name my-be-app be-app:latest
```

> **注意**：
> *   通常推荐使用 `docker-compose up --build`，因为它会自动处理网络连接和依赖服务（MySQL/Redis）。
> *   **单独运行**时，应用可能会因为连接不上 `mysql` 和 `redis` 主机名而报错（除非你配置了额外的 Docker 网络或修改了配置文件）。

## 5. 部署步骤

### 第一步：准备配置

确保 `conf/deploy.local.yml` 中的数据库连接配置与 `docker-compose.yaml` 保持一致：

```yaml
# conf/deploy.local.yml
mysql:
  ip: "mysql"         # 容器服务名
  username: "root"
  password: "root"

redis:
  ip: "redis"         # 容器服务名
```

### 第二步：编译与启动

在项目根目录 (`be/`) 下执行以下命令。该命令会自动完成以下操作：
1.  **构建镜像**：使用 Golang 镜像编译代码，生成最小化运行镜像。
2.  **启动服务**：启动 MySQL、Redis 和应用容器。
3.  **网络互联**：将所有容器加入 `be_app_net` 网络。

```bash
docker-compose up -d --build
```

### 第三步：初始化数据库

服务启动后，MySQL 容器 (`doing_now_mysql_compose`) 会自动运行，但表结构需要手动导入。

1.  **确认 MySQL 容器已启动**：
    ```bash
    docker ps
    ```

2.  **创建数据库表**：
    执行以下命令，将 `init.sql` 导入到 MySQL 容器中：

    ```bash
    # 读取 docs/sql/init.sql 并导入
    docker exec -i doing_now_mysql_compose mysql -uroot -proot doing_now < docs/sql/init.sql
    ```

### 第四步：验证部署

1.  **检查应用日志**：
    ```bash
    docker logs -f doing_now_app
    ```
    看到 `HTTP server listening on address=[::]:8000` 即表示启动成功。

2.  **验证数据库**：
    ```bash
    docker exec -it doing_now_mysql_compose mysql -uroot -proot doing_now -e "SHOW TABLES;"
    ```
    应输出 `users` 表。

## 6. 常用维护命令

*   **停止服务**：
    ```bash
    docker-compose down
    ```

*   **停止并清理数据**（慎用，会删除数据库数据）：
    ```bash
    docker-compose down -v
    ```

*   **查看日志**：
    ```bash
    docker-compose logs -f
    ```

*   **进入 MySQL 命令行**：
    ```bash
    docker exec -it doing_now_mysql_compose mysql -uroot -proot doing_now
    ```
