## Cấu hình hệ thống

Hệ thống được triển khai trên 3 VM với cấu hình như sau:

### VM 1 (~1.85 vCPUs, ~3.63GB RAM, ~13GB disk)
- Traefik: 0.1 vCPU, 128MB
- RabbitMQ: 0.25 vCPU, 512MB, ~1GB disk
- MySQL: 0.5 vCPU, 1GB, ~5GB disk
- MongoDB (retrieval-replica): 0.25 vCPU, 512MB, ~1GB disk
- create-service-1: 0.25 vCPU, 512MB
- retrieval-service-1: 0.5 vCPU, 1GB (~500 views/phút)

### VM 2 (~1.75 vCPUs, ~3.5GB RAM, ~9GB disk)
- MongoDB (retrieval-primary): 0.5 vCPU, 1GB, ~5GB disk
- create-service-2: 0.25 vCPU, 512MB
- retrieval-service-2: 0.75 vCPU, 1.5GB (~500 views/phút)
- Redis: 0.25 vCPU, 512MB, ~1GB disk

### VM 3 (~1.75 vCPUs, ~3.5GB RAM, ~11GB disk)
- Prometheus: 0.25 vCPU, 512MB, ~1GB disk
- Grafana: 0.25 vCPU, 512MB, ~1GB disk
- MongoDB (cleanup): 0.25 vCPU, 512MB, ~1GB disk
- cleanup-service: 0.25 vCPU, 512MB
- MongoDB (analytics): 0.5 vCPU, 1GB, ~5GB disk
- analytics-service: 0.25 vCPU, 512MB

## Thiết lập Docker Swarm

### 1. Khởi tạo Docker Swarm trên node đầu tiên (VM 1)

```bash
# Đăng nhập vào VM 1
ssh user@vm1_ip

# Khởi tạo swarm
sudo docker swarm init --advertise-addr <VM1_IP>
```

Sau khi chạy lệnh này, sẽ nhận được một đoạn mã lệnh dùng để thêm các node khác vào swarm, ví dụ:

```
docker swarm join --token SWMTKN-1-xxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxx <VM1_IP>:2377
```

### 2. Thêm các node khác vào swarm

Đăng nhập vào VM 2 và VM 3, chạy lệnh join nhận được từ bước trước:

```bash
# Đăng nhập vào VM 2
ssh user@vm2_ip

# Thêm vào swarm
sudo docker swarm join --token SWMTKN-1-xxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxx <VM1_IP>:2377

# Đăng nhập vào VM 3
ssh user@vm3_ip

# Thêm vào swarm
sudo docker swarm join --token SWMTKN-1-xxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxx <VM1_IP>:2377
```

### 3. Xác nhận các node đã tham gia swarm

Trên VM 1 (node manager), kiểm tra danh sách các node:

```bash
sudo docker node ls
```

Kết quả sẽ hiển thị cả 3 node với vai trò tương ứng (leader và worker).

## Tạo network

Tạo overlay network cho các dịch vụ trong swarm:

```bash
sudo docker network create --driver overlay --attachable pastebin-net
```

## Sửa file infrastructure/database/application.yml

Sửa các phần deploy trong các file `infrastructure/database/application.yml` ở thư mục deploy để phân bổ các dịch vụ

```yaml
version: '3.8'

services:
  create-service-1:
    image: your-repository/create-service:latest
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.hostname == vm1
      resources:
        limits:
          cpus: '0.25'
          memory: 512M
```

## Triển khai stack
Chạy file deploy

```bash
cd deploy/
./deploy.sh
```
