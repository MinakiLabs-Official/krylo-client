package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const serverURL = "https://shella.minaki.io"

// ─── styles ───────────────────────────────────────────────────────────────────
var (
	styleBanner   = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	styleTitle    = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	stylePrompt   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleUsername = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	styleTime     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleUpvote   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleBar      = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	styleRoom     = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	styleJoined   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleFollowed = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	stylePost = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1).MarginBottom(1)

	styleSelected = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("213")).
			Padding(0, 1).MarginBottom(1)

	styleCompose = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("213")).
			Padding(0, 1).Width(60)

	styleCard = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1).MarginBottom(1)

	styleCardSel = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("213")).
			Padding(0, 1).MarginBottom(1)
)

// ─── screens ──────────────────────────────────────────────────────────────────
type screen int

const (
	screenLogin screen = iota
	screenRegister
	screenFeed
	screenFollowingFeed
	screenRooms
	screenRoomFeed
	screenUsers
	screenInbox
	screenCompose
	screenDMCompose
)

// ─── types ────────────────────────────────────────────────────────────────────
type Post struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Username  string  `json:"username"`
	RoomID    *string `json:"room_id"`
	RoomName  *string `json:"room_name"`
	Body      string  `json:"body"`
	Upvotes   int     `json:"upvotes"`
	CreatedAt string  `json:"created_at"`
}

type Room struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberCount int    `json:"member_count"`
	Joined      bool   `json:"joined"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	Following bool   `json:"following"`
}

type Message struct {
	ID        string `json:"id"`
	FromID    string `json:"from_id"`
	FromUser  string `json:"from_username"`
	ToID      string `json:"to_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// ─── tea messages ─────────────────────────────────────────────────────────────
type authDoneMsg struct{ token, username, userID string }
type feedLoadedMsg struct{ posts []Post }
type roomsLoadedMsg struct{ rooms []Room }
type roomFeedLoadedMsg struct{ posts []Post }
type usersLoadedMsg struct{ users []User }
type inboxLoadedMsg struct{ messages []Message }
type postDoneMsg struct{}
type dmDoneMsg struct{}
type followDoneMsg struct{ userID string; following bool }
type joinDoneMsg struct{ roomID string; joined bool }
type errMsg struct{ err string }

// ─── model ────────────────────────────────────────────────────────────────────
type model struct {
	screen      screen
	prevScreen  screen
	inputs      []textinput.Model
	focused     int
	token       string
	username    string
	userID      string
	posts       []Post
	rooms       []Room
	users       []User
	messages    []Message
	cursor      int
	err         string
	loading     bool
	compose     textinput.Model
	dmInput     textinput.Model
	dmTarget    User
	activeRoom  Room
	width       int
	height      int
}

func initialModel() model {
	un := textinput.New()
	un.Placeholder = "username"
	un.Focus()
	un.Width = 30

	pw := textinput.New()
	pw.Placeholder = "password"
	pw.EchoMode = textinput.EchoPassword
	pw.Width = 30

	compose := textinput.New()
	compose.Placeholder = "what's on your mind?"
	compose.Width = 56

	dm := textinput.New()
	dm.Placeholder = "write a message..."
	dm.Width = 56

	return model{
		screen:  screenLogin,
		inputs:  []textinput.Model{un, pw},
		compose: compose,
		dmInput: dm,
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

// ─── api ──────────────────────────────────────────────────────────────────────
func apiGet(path, token string) ([]byte, error) {
	req, _ := http.NewRequest("GET", serverURL+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func apiPost(path, token string, payload interface{}) (*http.Response, error) {
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", serverURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

func doAuth(endpoint, username, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := apiPost(endpoint, "", map[string]string{"username": username, "password": password})
		if err != nil {
			return errMsg{"cannot reach server"}
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(resp.Body)
			return errMsg{string(b)}
		}
		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		return authDoneMsg{token: result["token"], username: result["username"], userID: result["id"]}
	}
}

func loadFeed(token string) tea.Cmd {
	return func() tea.Msg {
		b, err := apiGet("/api/feed", token)
		if err != nil {
			return errMsg{"cannot reach server"}
		}
		var posts []Post
		json.Unmarshal(b, &posts)
		return feedLoadedMsg{posts}
	}
}

func loadFollowingFeed(token string) tea.Cmd {
	return func() tea.Msg {
		b, err := apiGet("/api/feed/following", token)
		if err != nil {
			return errMsg{"cannot reach server"}
		}
		var posts []Post
		json.Unmarshal(b, &posts)
		return feedLoadedMsg{posts}
	}
}

func loadRooms(token string) tea.Cmd {
	return func() tea.Msg {
		b, err := apiGet("/api/rooms", token)
		if err != nil {
			return errMsg{"cannot reach server"}
		}
		var rooms []Room
		json.Unmarshal(b, &rooms)
		return roomsLoadedMsg{rooms}
	}
}

func loadRoomFeed(roomID, token string) tea.Cmd {
	return func() tea.Msg {
		b, err := apiGet("/api/rooms/"+roomID+"/feed", token)
		if err != nil {
			return errMsg{"cannot reach server"}
		}
		var posts []Post
		json.Unmarshal(b, &posts)
		return roomFeedLoadedMsg{posts}
	}
}

func loadUsers(token string) tea.Cmd {
	return func() tea.Msg {
		b, err := apiGet("/api/users", token)
		if err != nil {
			return errMsg{"cannot reach server"}
		}
		var users []User
		json.Unmarshal(b, &users)
		return usersLoadedMsg{users}
	}
}

func loadInbox(token string) tea.Cmd {
	return func() tea.Msg {
		b, err := apiGet("/api/inbox", token)
		if err != nil {
			return errMsg{"cannot reach server"}
		}
		var msgs []Message
		json.Unmarshal(b, &msgs)
		return inboxLoadedMsg{msgs}
	}
}

func submitPost(token, body string, roomID *string) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]interface{}{"body": body, "room_id": roomID}
		resp, err := apiPost("/api/post", token, payload)
		if err != nil || resp.StatusCode != 201 {
			return errMsg{"post failed"}
		}
		return postDoneMsg{}
	}
}

func sendDM(token, toID, body string) tea.Cmd {
	return func() tea.Msg {
		resp, err := apiPost("/api/message", token, map[string]string{"to_id": toID, "body": body})
		if err != nil || resp.StatusCode != 201 {
			return errMsg{"dm failed"}
		}
		return dmDoneMsg{}
	}
}

func doFollow(token, userID string, unfollow bool) tea.Cmd {
	return func() tea.Msg {
		path := "/api/follow/" + userID
		if unfollow {
			path = "/api/unfollow/" + userID
		}
		apiPost(path, token, nil)
		return followDoneMsg{userID: userID, following: !unfollow}
	}
}

func doJoin(token, roomID string, leave bool) tea.Cmd {
	return func() tea.Msg {
		path := "/api/rooms/" + roomID + "/join"
		if leave {
			path = "/api/rooms/" + roomID + "/leave"
		}
		apiPost(path, token, nil)
		return joinDoneMsg{roomID: roomID, joined: !leave}
	}
}

func doUpvote(token, postID string) tea.Cmd {
	return func() tea.Msg {
		apiPost("/api/upvote/"+postID, token, nil)
		return nil
	}
}

// ─── update ───────────────────────────────────────────────────────────────────
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case authDoneMsg:
		m.token, m.username, m.userID = msg.token, msg.username, msg.userID
		m.loading, m.err = false, ""
		return m, loadFeed(m.token)

	case feedLoadedMsg:
		m.posts = msg.posts
		m.screen = screenFeed
		if m.prevScreen == screenFollowingFeed {
			m.screen = screenFollowingFeed
		}
		m.loading, m.cursor = false, 0
		return m, nil

	case roomsLoadedMsg:
		m.rooms = msg.rooms
		m.screen = screenRooms
		m.loading, m.cursor = false, 0
		return m, nil

	case roomFeedLoadedMsg:
		m.posts = msg.posts
		m.screen = screenRoomFeed
		m.loading, m.cursor = false, 0
		return m, nil

	case usersLoadedMsg:
		m.users = msg.users
		m.screen = screenUsers
		m.loading, m.cursor = false, 0
		return m, nil

	case inboxLoadedMsg:
		m.messages = msg.messages
		m.screen = screenInbox
		m.loading = false
		return m, nil

	case postDoneMsg:
		m.compose.SetValue("")
		m.loading = false
		if m.screen == screenCompose {
			if m.prevScreen == screenRoomFeed {
				m.screen = screenRoomFeed
				return m, loadRoomFeed(m.activeRoom.ID, m.token)
			}
			m.prevScreen = screenFeed
			return m, loadFeed(m.token)
		}
		return m, loadFeed(m.token)

	case dmDoneMsg:
		m.dmInput.SetValue("")
		m.screen = screenUsers
		m.loading = false
		return m, loadUsers(m.token)

	case followDoneMsg:
		for i, u := range m.users {
			if u.ID == msg.userID {
				m.users[i].Following = msg.following
			}
		}
		m.loading = false
		return m, nil

	case joinDoneMsg:
		for i, r := range m.rooms {
			if r.ID == msg.roomID {
				m.rooms[i].Joined = msg.joined
			}
		}
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch m.screen {
		case screenLogin, screenRegister:
			return m.updateAuth(msg)
		case screenFeed, screenFollowingFeed:
			return m.updateFeed(msg)
		case screenRooms:
			return m.updateRooms(msg)
		case screenRoomFeed:
			return m.updateRoomFeed(msg)
		case screenUsers:
			return m.updateUsers(msg)
		case screenInbox:
			return m.updateInbox(msg)
		case screenCompose:
			return m.updateCompose(msg)
		case screenDMCompose:
			return m.updateDMCompose(msg)
		}
	}
	return m, nil
}

func (m model) updateAuth(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "tab", "down":
		m.focused = (m.focused + 1) % 2
	case "up":
		m.focused = (m.focused - 1 + 2) % 2
	case "enter":
		if m.focused == 1 {
			m.loading = true
			m.err = ""
			un, pw := m.inputs[0].Value(), m.inputs[1].Value()
			if m.screen == screenLogin {
				return m, doAuth("/api/login", un, pw)
			}
			return m, doAuth("/api/register", un, pw)
		}
		m.focused = 1
	case "ctrl+r":
		if m.screen == screenLogin {
			m.screen = screenRegister
		} else {
			m.screen = screenLogin
		}
		m.err = ""
	}
	for i := range m.inputs {
		if i == m.focused {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) updateFeed(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.posts)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "r":
		m.loading = true
		return m, loadFeed(m.token)
	case "n":
		m.prevScreen = screenFeed
		m.screen = screenCompose
		m.compose.Focus()
		return m, textinput.Blink
	case "v":
		if len(m.posts) > 0 {
			return m, doUpvote(m.token, m.posts[m.cursor].ID)
		}
	case "R":
		m.loading = true
		m.prevScreen = screenRooms
		return m, loadRooms(m.token)
	case "u":
		m.loading = true
		return m, loadUsers(m.token)
	case "i":
		m.loading = true
		return m, loadInbox(m.token)
	case "f":
		m.loading = true
		m.prevScreen = screenFollowingFeed
		return m, loadFollowingFeed(m.token)
	case "g":
		m.loading = true
		m.prevScreen = screenFeed
		return m, loadFeed(m.token)
	}
	return m, nil
}

func (m model) updateRooms(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.rooms)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(m.rooms) > 0 {
			m.activeRoom = m.rooms[m.cursor]
			m.loading = true
			return m, loadRoomFeed(m.activeRoom.ID, m.token)
		}
	case "j2", "space":
		if len(m.rooms) > 0 {
			r := m.rooms[m.cursor]
			return m, doJoin(m.token, r.ID, r.Joined)
		}
	case "esc", "b":
		m.prevScreen = screenFeed
		return m, loadFeed(m.token)
	case "g":
		m.loading = true
		m.prevScreen = screenFeed
		return m, loadFeed(m.token)
	case "u":
		m.loading = true
		return m, loadUsers(m.token)
	case "i":
		m.loading = true
		return m, loadInbox(m.token)
	}
	return m, nil
}

func (m model) updateRoomFeed(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.posts)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "n":
		m.prevScreen = screenRoomFeed
		m.screen = screenCompose
		m.compose.Focus()
		return m, textinput.Blink
	case "v":
		if len(m.posts) > 0 {
			return m, doUpvote(m.token, m.posts[m.cursor].ID)
		}
	case "r":
		m.loading = true
		return m, loadRoomFeed(m.activeRoom.ID, m.token)
	case "esc", "b":
		m.loading = true
		return m, loadRooms(m.token)
	case "g":
		m.loading = true
		m.prevScreen = screenFeed
		return m, loadFeed(m.token)
	case "u":
		m.loading = true
		return m, loadUsers(m.token)
	case "i":
		m.loading = true
		return m, loadInbox(m.token)
	}
	return m, nil
}

func (m model) updateUsers(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.users)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "f":
		if len(m.users) > 0 {
			u := m.users[m.cursor]
			if u.ID != m.userID {
				return m, doFollow(m.token, u.ID, u.Following)
			}
		}
	case "d":
		if len(m.users) > 0 {
			u := m.users[m.cursor]
			if u.ID != m.userID {
				m.dmTarget = u
				m.screen = screenDMCompose
				m.dmInput.Focus()
				return m, textinput.Blink
			}
		}
	case "esc", "b", "g":
		m.loading = true
		m.prevScreen = screenFeed
		return m, loadFeed(m.token)
	case "i":
		m.loading = true
		return m, loadInbox(m.token)
	case "R":
		m.loading = true
		return m, loadRooms(m.token)
	}
	return m, nil
}

func (m model) updateInbox(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b", "g":
		m.loading = true
		m.prevScreen = screenFeed
		return m, loadFeed(m.token)
	case "r":
		m.loading = true
		return m, loadInbox(m.token)
	case "u":
		m.loading = true
		return m, loadUsers(m.token)
	}
	return m, nil
}

func (m model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = m.prevScreen
		m.compose.Blur()
		m.err = ""
		return m, nil
	case "enter":
		if m.compose.Value() != "" {
			m.loading = true
			body := m.compose.Value()
			var roomID *string
			if m.prevScreen == screenRoomFeed {
				id := m.activeRoom.ID
				roomID = &id
			}
			return m, submitPost(m.token, body, roomID)
		}
	}
	var cmd tea.Cmd
	m.compose, cmd = m.compose.Update(msg)
	return m, cmd
}

func (m model) updateDMCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenUsers
		m.dmInput.Blur()
		m.err = ""
		return m, nil
	case "enter":
		if m.dmInput.Value() != "" {
			m.loading = true
			body := m.dmInput.Value()
			return m, sendDM(m.token, m.dmTarget.ID, body)
		}
	}
	var cmd tea.Cmd
	m.dmInput, cmd = m.dmInput.Update(msg)
	return m, cmd
}

// ─── view ─────────────────────────────────────────────────────────────────────
const banner = `  ███████╗██╗  ██╗███████╗██╗     ██╗      █████╗ 
  ██╔════╝██║  ██║██╔════╝██║     ██║     ██╔══██╗
  ███████╗███████║█████╗  ██║     ██║     ███████║
  ╚════██║██╔══██║██╔══╝  ██║     ██║     ██╔══██║
  ███████║██║  ██║███████╗███████╗███████╗██║  ██║
  ╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝`

func bar() string {
	return styleBar.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func header(m model, subtitle string) string {
	return styleBanner.Render("  ◈ shella") + "  " +
		styleUsername.Render("@"+m.username) + "  " +
		styleTime.Render(subtitle) + "\n" + bar() + "\n\n"
}

func (m model) View() string {
	switch m.screen {
	case screenLogin:
		return m.viewAuth("sign in to shella", "ctrl+r → register")
	case screenRegister:
		return m.viewAuth("create account", "ctrl+r → back to login")
	case screenFeed:
		return m.viewPosts("global feed", "g global  f following  R rooms  u users  i inbox  n post  v upvote  r refresh  q quit")
	case screenFollowingFeed:
		return m.viewPosts("following", "g global  f following  R rooms  u users  i inbox  n post  v upvote  r refresh  q quit")
	case screenRooms:
		return m.viewRooms()
	case screenRoomFeed:
		return m.viewRoomFeed()
	case screenUsers:
		return m.viewUsers()
	case screenInbox:
		return m.viewInbox()
	case screenCompose:
		return m.viewCompose()
	case screenDMCompose:
		return m.viewDMCompose()
	}
	return ""
}

func (m model) viewAuth(title, hint string) string {
	s := styleBanner.Render(banner) + "\n\n"
	s += styleTitle.Render("  ┤ "+title+" ├") + "\n\n"
	s += "  " + m.inputs[0].View() + "\n"
	s += "  " + m.inputs[1].View() + "\n\n"
	s += styleHelp.Render("  enter → submit  │  tab → switch  │  "+hint)
	if m.err != "" {
		s += "\n\n  " + styleError.Render("✗ "+m.err)
	}
	if m.loading {
		s += "\n\n  " + stylePrompt.Render("connecting...")
	}
	return s
}

func (m model) viewPosts(title, help string) string {
	s := header(m, title)
	if m.loading {
		return s + stylePrompt.Render("  loading...") + "\n"
	}
	if len(m.posts) == 0 {
		s += stylePrompt.Render("  no posts yet — press n to write the first one") + "\n"
	}
	for i, p := range m.posts {
		t, _ := time.Parse(time.RFC3339Nano, p.CreatedAt)
		timeStr := t.Format("Jan 02 15:04")
		roomTag := ""
		if p.RoomName != nil {
			roomTag = "  " + styleRoom.Render("#"+*p.RoomName)
		}
		header2 := styleUsername.Render("@"+p.Username) +
			styleTime.Render("  "+timeStr) +
			roomTag + "  " +
			styleUpvote.Render(fmt.Sprintf("▲ %d", p.Upvotes))
		content := header2 + "\n" + p.Body
		if i == m.cursor {
			s += styleSelected.Render(content) + "\n"
		} else {
			s += stylePost.Render(content) + "\n"
		}
	}
	s += bar() + "\n"
	s += styleHelp.Render("  " + help)
	if m.err != "" {
		s += "\n  " + styleError.Render("✗ "+m.err)
	}
	return s
}

func (m model) viewRooms() string {
	s := header(m, "rooms")
	if m.loading {
		return s + stylePrompt.Render("  loading...") + "\n"
	}
	for i, r := range m.rooms {
		joinedTag := ""
		if r.Joined {
			joinedTag = "  " + styleJoined.Render("✓ joined")
		}
		members := styleDim.Render(fmt.Sprintf("%d members", r.MemberCount))
		line := styleRoom.Render("#"+r.Name) + "  " + members + joinedTag + "\n" +
			styleDim.Render(r.Description)
		if i == m.cursor {
			s += styleCardSel.Render(line) + "\n"
		} else {
			s += styleCard.Render(line) + "\n"
		}
	}
	s += bar() + "\n"
	s += styleHelp.Render("  enter → open  │  space → join/leave  │  j/k → navigate  │  g → feed  │  u → users  │  q → quit")
	return s
}

func (m model) viewRoomFeed() string {
	title := styleRoom.Render("#"+m.activeRoom.Name) + styleTime.Render("  "+m.activeRoom.Description)
	s := header(m, title)
	if m.loading {
		return s + stylePrompt.Render("  loading...") + "\n"
	}
	if len(m.posts) == 0 {
		s += stylePrompt.Render("  no posts in #"+m.activeRoom.Name+" yet — press n to post") + "\n"
	}
	for i, p := range m.posts {
		t, _ := time.Parse(time.RFC3339Nano, p.CreatedAt)
		timeStr := t.Format("Jan 02 15:04")
		h := styleUsername.Render("@"+p.Username) +
			styleTime.Render("  "+timeStr) + "  " +
			styleUpvote.Render(fmt.Sprintf("▲ %d", p.Upvotes))
		content := h + "\n" + p.Body
		if i == m.cursor {
			s += styleSelected.Render(content) + "\n"
		} else {
			s += stylePost.Render(content) + "\n"
		}
	}
	s += bar() + "\n"
	s += styleHelp.Render("  n → post  │  v → upvote  │  r → refresh  │  b → back to rooms  │  g → global feed  │  q → quit")
	return s
}

func (m model) viewUsers() string {
	s := header(m, "users")
	if m.loading {
		return s + stylePrompt.Render("  loading...") + "\n"
	}
	for i, u := range m.users {
		followTag := ""
		if u.Following {
			followTag = "  " + styleFollowed.Render("★ following")
		}
		youTag := ""
		if u.ID == m.userID {
			youTag = "  " + styleDim.Render("(you)")
		}
		bio := u.Bio
		if bio == "" {
			bio = "no bio yet"
		}
		line := styleUsername.Render("@"+u.Username) + youTag + followTag + "\n" +
			styleDim.Render(bio)
		if i == m.cursor {
			s += styleCardSel.Render(line) + "\n"
		} else {
			s += styleCard.Render(line) + "\n"
		}
	}
	s += bar() + "\n"
	s += styleHelp.Render("  f → follow/unfollow  │  d → DM  │  j/k → navigate  │  g → feed  │  i → inbox  │  q → quit")
	return s
}

func (m model) viewInbox() string {
	s := header(m, "inbox")
	if m.loading {
		return s + stylePrompt.Render("  loading...") + "\n"
	}
	if len(m.messages) == 0 {
		s += stylePrompt.Render("  no messages yet") + "\n"
	}
	for _, msg := range m.messages {
		t, _ := time.Parse(time.RFC3339Nano, msg.CreatedAt)
		timeStr := t.Format("Jan 02 15:04")
		line := styleUsername.Render("@" + msg.FromUser) +
			styleTime.Render("  "+timeStr) + "\n" +
			msg.Body
		s += styleCard.Render(line) + "\n"
	}
	s += bar() + "\n"
	s += styleHelp.Render("  r → refresh  │  u → users  │  g → feed  │  q → quit")
	return s
}

func (m model) viewCompose() string {
	roomTag := ""
	if m.prevScreen == screenRoomFeed {
		roomTag = " in " + styleRoom.Render("#"+m.activeRoom.Name)
	}
	s := header(m, "new post"+roomTag)
	s += "  " + styleCompose.Render(m.compose.View()) + "\n\n"
	s += styleHelp.Render("  enter → post  │  esc → back")
	if m.loading {
		s += "\n  " + stylePrompt.Render("posting...")
	}
	if m.err != "" {
		s += "\n  " + styleError.Render("✗ "+m.err)
	}
	return s
}

func (m model) viewDMCompose() string {
	s := header(m, "dm → "+m.dmTarget.Username)
	s += "  " + styleCompose.Render(m.dmInput.View()) + "\n\n"
	s += styleHelp.Render("  enter → send  │  esc → back")
	if m.loading {
		s += "\n  " + stylePrompt.Render("sending...")
	}
	if m.err != "" {
		s += "\n  " + styleError.Render("✗ "+m.err)
	}
	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
