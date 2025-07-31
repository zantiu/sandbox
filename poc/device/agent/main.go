package main

import "go.uber.org/zap"

func main() {
	// Initialize the logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	sugar := logger.Sugar()

	config := &Config{
		DeviceID:  "test-device",
		WfmSbiUrl: "http://localhost:3000/margo/sbi",
	}

	agent, err := NewDeviceAgent(config, sugar)
	if err != nil {
		sugar.Fatalf("Failed to create device agent: %v", err)
	}

	if err := agent.Start(); err != nil {
		sugar.Fatalf("Failed to start device agent: %v", err)
	}

	select {}
}
