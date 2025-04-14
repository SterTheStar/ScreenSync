# 🌸 ScreenSync

ScreenSync é uma ferramenta de monitoramento de sistema que captura screenshots e informações do sistema periodicamente e as envia para um canal do Discord através de webhooks.

## ✨ Funcionalidades

- 📸 Captura de tela automática a cada 5 segundos
- 💻 Monitoramento de informações do sistema em tempo real:
  - Informações do CPU e uso atual
  - Memória RAM total e em uso
  - Informações da GPU e VRAM
  - Uso de disco e partições
  - Estatísticas de rede (upload/download)
  - Top 5 processos por uso de CPU
  - Uptime do sistema

## 🚀 Como usar

1. Clone este repositório
2. Configure sua URL do webhook do Discord no arquivo `main.go`:
   ```go
   const webhookURL = "seu_webhook_aqui"
   ```
3. Execute o programa:
   ```bash
   go build
   ./screensync # ou screensync.exe no Windows
   ```

## ⚙️ Requisitos

- Go 1.24.2 ou superior
- Sistema operacional Windows
- Webhook do Discord configurado

## 📦 Dependências

Bibliotecas principais:
```go
"github.com/StackExchange/wmi"         // Informações da GPU no Windows
"github.com/kbinani/screenshot"        // Captura de tela
"github.com/shirou/gopsutil/v3/..."   // Informações do sistema
```

## ⚠️ Aviso Legal

Este projeto é fornecido APENAS para fins educacionais e de estudo. 

- ❌ NÃO use para monitoramento não autorizado
- ❌ NÃO use em ambientes de produção
- ✅ SEMPRE obtenha as permissões necessárias
- ✅ USE apenas para aprendizado e desenvolvimento

## 📝 Licença

Este projeto está sob a licença MIT com avisos adicionais de não responsabilização. 
Veja o arquivo [LICENSE](LICENSE) para mais detalhes.