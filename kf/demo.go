package kf

//
//import (
//	"context"
//	"fmt"
//	"github.com/gin-gonic/gin"
//	kf "github.com/open4go/p7/kf"
//	"time"
//)
//
//func main() {
//	// 初始化 Kafka Producer 和 Consumer
//	kf.InitWriter([]string{"localhost:19092"}, "demo-input")
//	kf.InitReader([]string{"localhost:19092"}, "demo-input", "demo-group")
//
//	defer kf.CloseWriter()
//	defer kf.CloseReader()
//
//	// 启动后台消费者
//	kf.ConsumeLoop(context.Background(), func(data []byte) {
//		fmt.Printf("🔄 Received message: %s\n", string(data))
//	})
//
//	// Gin HTTP API
//	r := gin.Default()
//
//	r.GET("/send/:msg", func(c *gin.Context) {
//		msg := c.Param("msg")
//		err := kafka.SendString(context.Background(), msg)
//		if err != nil {
//			c.JSON(500, gin.H{"error": err.Error()})
//			return
//		}
//		c.JSON(200, gin.H{"sent": msg})
//	})
//
//	r.POST("/send-bytes", func(c *gin.Context) {
//		data, err := c.GetRawData()
//		if err != nil {
//			c.JSON(400, gin.H{"error": err.Error()})
//			return
//		}
//		err = kafka.SendBytes(context.Background(), data)
//		if err != nil {
//			c.JSON(500, gin.H{"error": err.Error()})
//			return
//		}
//		c.JSON(200, gin.H{"sent-bytes": len(data)})
//	})
//
//	r.Run(":8080")
//}
