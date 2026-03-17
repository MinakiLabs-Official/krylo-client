# 🐚 shella

> the terminal you actually want to live in

A social network that lives in your terminal. Feed, rooms, DMs, follows — no browser required.
```
  ███████╗██╗  ██╗███████╗██╗     ██╗      █████╗ 
  ██╔════╝██║  ██║██╔════╝██║     ██║     ██╔══██╗
  ███████╗███████║█████╗  ██║     ██║     ███████║
  ╚════██║██╔══██║██╔══╝  ██║     ██║     ██╔══██║
  ███████║██║  ██║███████╗███████╗███████╗██║  ██║
  ╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝
```

## install

### linux
```bash
curl -sL https://shella.minaki.io/install.sh | sh
```

### windows (WSL)
```bash
curl -sL https://shella.minaki.io/install.sh | sh
```

### build from source
```bash
git clone https://github.com/MinakiLabs-Official/shella-client
cd shella-client
go run main.go
```

## navigation

| key | action |
|-----|--------|
| `g` | global feed |
| `f` | following feed |
| `R` | rooms |
| `u` | users |
| `i` | inbox (DMs) |
| `n` | new post |
| `v` | upvote |
| `d` | DM someone |
| `j/k` | navigate up/down |
| `esc` | go back |
| `q` | quit |

## server

`shella` runs on [shella.minaki.io](https://shella.minaki.io)

Built by [MinakiLabs](https://minaki.io)
