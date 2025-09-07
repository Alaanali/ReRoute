package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Alaanali/ReRoute/colors"
)

func getStatusColor(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return colors.Green
	case statusCode >= 300 && statusCode < 400:
		return colors.Yellow
	case statusCode >= 400 && statusCode < 500:
		return colors.Red
	case statusCode >= 500:
		return colors.Purple
	default:
		return colors.White
	}
}

func getMethodColor(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return colors.Blue
	case "POST":
		return colors.Green
	case "PUT":
		return colors.Yellow
	case "DELETE":
		return colors.Red
	case "PATCH":
		return colors.Purple
	default:
		return colors.White
	}
}

func (c *Client) printTunnelInfo() {
	tunnelURL := fmt.Sprintf("http://%s.%s:8000", c.Id, c.tunnelHost)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%s %s\n",
		colors.Colorize(colors.Green+colors.Bold, "🚀 Tunnel Active:"),
		colors.Colorize(colors.Cyan+colors.Bold, tunnelURL))
	fmt.Printf("%s %s\n",
		colors.Colorize(colors.Blue, "📡 Forwarding to:"),
		colors.Colorize(colors.Yellow, fmt.Sprintf("localhost:%s", c.localhostPort)))
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%s\n", colors.Colorize(colors.White, "Request Log:"))
}

func (c *Client) printRequest(response *http.Response, request *http.Request, duration time.Duration) {
	statusColor := getStatusColor(response.StatusCode)
	methodColor := getMethodColor(request.Method)

	var icon string
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		icon = colors.Colorize(colors.Green, "✓")
	} else {
		icon = colors.Colorize(colors.Red, "✗")
	}

	timestamp := time.Now().Format("15:04:05")
	timestampStr := colors.Colorize(colors.White, fmt.Sprintf("[%s]", timestamp))

	fmt.Printf("%s %s %s %s %s %s\n",
		timestampStr,
		icon,
		colors.Colorize(methodColor, fmt.Sprintf("%-6s", request.Method)),
		colors.Colorize(statusColor, fmt.Sprintf("%-3d", response.StatusCode)),
		request.URL.Path,
		colors.Colorize(colors.White, fmt.Sprintf("(%v)", duration.Round(time.Millisecond))),
	)
}
