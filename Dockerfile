FROM alpine:latest

# Cài đặt chứng chỉ để gọi được https và múi giờ
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy file 'main' từ máy thật vào container
COPY main .

EXPOSE 8080

# Chạy file main
CMD ["./main"]