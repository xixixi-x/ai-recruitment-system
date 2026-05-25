package main

import (
	"context"
	"log"
	"net"

	"final_homework/logic-grpc-service/internal/ai"
	"final_homework/logic-grpc-service/internal/config"
	"final_homework/logic-grpc-service/internal/grpcjson"
	"final_homework/logic-grpc-service/internal/logic"
	"final_homework/logic-grpc-service/internal/model"
	"final_homework/logic-grpc-service/internal/ossstore"
	"final_homework/logic-grpc-service/internal/rpc"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load("../.env", ".env")
	cfg := config.Load()

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect mysql failed: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Job{}, &model.CandidateProfile{}, &model.Application{}, &model.AIChatMessage{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	ossClient, err := ossstore.New(cfg.OSSEndpoint, cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret, cfg.OSSBucket, cfg.OSSSignExpire)
	if err != nil {
		log.Fatalf("init private oss failed: %v", err)
	}

	assistant, err := ai.New(context.Background(), cfg.AIAPIKey, cfg.AIBaseURL, cfg.AIModel)
	if err != nil {
		log.Fatalf("init eino assistant failed: %v", err)
	}

	encoding.RegisterCodec(grpcjson.Codec{})
	server := grpc.NewServer()
	rpc.RegisterLogicServiceServer(server, logic.New(db, ossClient, assistant))

	lis, err := net.Listen("tcp", ":"+cfg.LogicGRPCPort)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	log.Printf("logic-grpc-service listening on :%s", cfg.LogicGRPCPort)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("serve failed: %v", err)
	}
}
