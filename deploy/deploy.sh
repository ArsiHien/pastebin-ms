# Các màu để hiển thị thông báo
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Tạo mạng nếu chưa tồn tại
echo -e "${YELLOW}Kiểm tra và tạo mạng pastebin-net...${NC}"
if ! sudo docker network ls | grep -q pastebin-net; then
  sudo docker network create --driver overlay --attachable pastebin-net
  echo -e "${GREEN}Đã tạo mạng pastebin-net${NC}"
else
  echo -e "${GREEN}Mạng pastebin-net đã tồn tại${NC}"
fi

# Triển khai stack cơ sở hạ tầng (infrastructure)
echo -e "${YELLOW}Triển khai stack cơ sở hạ tầng...${NC}"
sudo docker stack deploy --with-registry-auth -c infrastructure.yml infrastructure
echo -e "${GREEN}Đã triển khai stack cơ sở hạ tầng${NC}"

# Đợi Traefik khởi động
echo -e "${YELLOW}Đợi Traefik khởi động...${NC}"
until sudo docker service ls | grep infrastructure_traefik | grep -q "1/1"; do
  echo "Đang đợi Traefik..."
  sleep 5
done
echo -e "${GREEN}Traefik đã sẵn sàng${NC}"

# Đợi rabbitmq khởi động
echo -e "${YELLOW}Đợi rabbitmq khởi động...${NC}"
until sudo docker service ls | grep infrastructure_rabbitmq | grep -q "1/1"; do
  echo "Đang đợi rabbitmq..."
  sleep 10
done
echo -e "${GREEN}rabbitmq đã sẵn sàng${NC}"

# Triển khai stack cơ sở dữ liệu
echo -e "${YELLOW}Triển khai stack cơ sở dữ liệu...${NC}"
sudo docker stack deploy --with-registry-auth -c database.yml database
echo -e "${GREEN}Đã triển khai stack cơ sở dữ liệu${NC}"

# Hàm kiểm tra trạng thái health của container
check_health() {
  local service=$1
  local container_id=$(sudo docker ps -q --filter "name=${service}")
  
  if [ -z "$container_id" ]; then
    echo -e "${RED}Không tìm thấy container cho service ${service}${NC}"
    return 1
  fi
  
  local health_status=$(sudo docker inspect --format='{{.State.Health.Status}}' $container_id 2>/dev/null)
  
  if [ -z "$health_status" ] || [ "$health_status" = "<nil>" ]; then
    echo -e "${YELLOW}Service ${service} không có health check${NC}"
    return 0
  elif [ "$health_status" = "healthy" ]; then
    echo -e "${GREEN}Service ${service} đã healthy${NC}"
    return 0
  else
    echo -e "${YELLOW}Service ${service} đang ${health_status}${NC}"
    return 1
  fi
}

# Đợi các dịch vụ cơ sở dữ liệu khởi động
echo -e "${YELLOW}Đợi các dịch vụ cơ sở dữ liệu khởi động...${NC}"

services=("database_mysql" "database_redis" "database_mongo-retrieval-1" "database_mongo-retrieval-2" "database_mongo-cleanup" "database_mongo-analytics")

for service in "${services[@]}"; do
  echo -e "${YELLOW}Đợi service ${service} khởi động...${NC}"
  
  # Đợi service được tạo
  until sudo docker service ls | grep -q "$service"; do
    echo "Đang đợi service ${service} được tạo..."
    sleep 5
  done
  
  # Đợi replica được triển khai đầy đủ
  until sudo docker service ls | grep "$service" | grep -q -E "1/1|2/2"; do
    echo "Đang đợi service ${service} triển khai đầy đủ..."
    sleep 5
  done
  
  # Lấy container ID để kiểm tra health
  container_name=$(sudo docker ps --filter "name=${service}" --format "{{.Names}}" | head -n 1)
  
  if [ -n "$container_name" ]; then
    # Đợi container healthy
    until check_health "$container_name"; do
      echo "Đang đợi ${container_name} healthy..."
      sleep 10
    done
  else
    echo -e "${YELLOW}Không tìm thấy container cho service ${service}, bỏ qua kiểm tra health${NC}"
  fi
  
  echo -e "${GREEN}Service ${service} đã sẵn sàng${NC}"
done

# Triển khai stack ứng dụng
echo -e "${YELLOW}Triển khai stack ứng dụng...${NC}"
sudo docker stack deploy --with-registry-auth -c application.yml application
echo -e "${GREEN}Đã triển khai stack ứng dụng${NC}"

# Kiểm tra trạng thái của tất cả các service
echo -e "${YELLOW}Kiểm tra trạng thái tất cả các service...${NC}"
sudo docker stack services infrastructure
sudo docker stack services database
sudo docker stack services application

echo -e "${GREEN}Hoàn tất triển khai!${NC}"