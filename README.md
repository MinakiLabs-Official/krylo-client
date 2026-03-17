# krylo

> the terminal you actually want to live in
```
  ██╗  ██╗██████╗ ██╗   ██╗██╗      ██████╗ 
  ██║ ██╔╝██╔══██╗╚██╗ ██╔╝██║     ██╔═══██╗
  █████╔╝ ██████╔╝ ╚████╔╝ ██║     ██║   ██║
  ██╔═██╗ ██╔══██╗  ╚██╔╝  ██║     ██║   ██║
  ██║  ██╗██║  ██║   ██║   ███████╗╚██████╔╝
  ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚══════╝ ╚═════╝ 
```

A social network that lives in your terminal. Feed, rooms, DMs, follows — no browser required.

## install

**Linux / Mac / WSL**
```bash
curl -sL https://krylo.minaki.io/install.sh | sh
```

**Windows**
Download [krylo.exe](https://krylo.minaki.io/dl/krylo.exe) and run it in [Windows Terminal](https://aka.ms/terminal).

**Build from source**
```bash
git clone https://github.com/MinakiLabs-Official/krylo-client
cd krylo-client
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
| `j/k` | navigate |
| `esc` | back |
| `q` | quit |

## server

Runs on [krylo.minaki.io](https://krylo.minaki.io) — built by [MinakiLabs](https://minaki.io)
