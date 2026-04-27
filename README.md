# Trading Bot V2

TradingView sinyallerini alip Binance Futures'ta otomatik islem yapan, 5 mikroservisli trading bot sistemi.

## Mimari

- **webhook-gateway** (Go) - TradingView webhook'larini alir, RabbitMQ'ya yayinlar
- **trading-engine** (Go) - Sinyalleri isler, Binance'e emir gonderir, PostgreSQL'e yazar
- **notification-worker** (Go) - Telegram bildirimleri gonderir
- **api-server** (Go) - Frontend icin REST API sunar
- **frontend** (React + TypeScript) - Dashboard, islem takibi, kar/zarar analizi

## Hizli Baslangic

1. `.env.example` dosyasini `.env` olarak kopyalayip degerleri doldurun:

```bash
cp .env.example .env
```

2. Docker Compose ile baslatin:

```bash
docker compose up -d
```

3. Erisim adresleri:
   - Frontend: http://localhost:3000
   - Webhook endpoint: http://localhost:8080/webhook
   - API: http://localhost:8081/api
   - RabbitMQ Management: http://localhost:15672

## TradingView Webhook Formati

```json
{
  "signal": "AL1",
  "ticker": "{{ticker}}",
  "secret": "optional_shared_secret"
}
```

Gecerli sinyal degerleri: `AL1`, `AL2`, `AL3`, `SAT1`, `SAT2`, `SAT3`

## Yapilandirma

Frontend uzerinden Ayarlar sayfasindan degistirilebilir:
- Islem tutari (USD)
- Kaldırac (1-125x)
- Margin modu (ISOLATED / CROSSED)
- Komisyon orani (varsayilan %0.04)
