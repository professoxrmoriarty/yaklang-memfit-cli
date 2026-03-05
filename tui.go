package aiclientv2

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------- tea.Msg ----------

type connectResultMsg struct {
	setting *SettingResponse
	err     error
}

type sessionCreatedMsg struct {
	runID string
	err   error
}

type sseStartedMsg struct {
	ch     <-chan SSEEvent
	cancel context.CancelFunc
}

type sseEventMsg SSEEvent
type sseDoneMsg struct{}

type sseErrorMsg struct{ err error }
type pushResultMsg struct{ err error }
type cancelResultMsg struct{ err error }
type providerOptionsLoadedMsg struct {
	options []slashCmd
	err     error
}
type modelOptionsLoadedMsg struct {
	options []slashCmd
	err     error
}

// ---------- phase ----------

type tuiPhase int

const (
	phaseConnecting tuiPhase = iota
	phaseReady
	phaseCreating
	phaseStreaming
	phaseInteractive
)

// ---------- styles ----------

var (
	sHeaderBar = lipgloss.NewStyle().Bold(true).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)
	sThought = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Italic(true)
	sTool    = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	sErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	sSystem  = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	sPrompt  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	sDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sInput   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	sLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	sModel   = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))

	sUserBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("208")).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(1)
	sInputBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("208")).
			PaddingTop(1).
			PaddingBottom(0).
			PaddingLeft(1)
)

// ---------- command palette ----------

type slashCmd struct {
	Name    string
	Desc    string
	SubOpts []slashCmd
}

var slashCommands = []slashCmd{
	{"/help", "显示可用命令帮助", nil},
	{"/setting", "查看当前 AI 设置", nil},
	{"/provider", "选择 Provider", nil},
	{"/model", "选择 Model", nil},
	{"/focus_mode", "选择 Focus Mode", nil},
	{"/set", "更新设置", []slashCmd{
		{"service", "设置 AI 服务提供商", nil},
		{"model", "设置使用的模型", nil},
		{"forge", "设置智能应用", nil},
		{"policy", "设置执行策略", nil},
		{"provider", "设置 provider id", nil},
		{"selected_model", "设置已选模型", nil},
		{"focus_mode", "设置 focus mode", nil},
	}},
	{"/review_policy", "设置审查策略", []slashCmd{
		{"auto", "自动审批 (含冷却间隔)", nil},
		{"ai", "AI 审批，AI 拒绝则转人工", nil},
		{"manual", "人工审批，逐步确认", nil},
		{"ai-auto", "AI 审批，AI 拒绝则自动放行", nil},
	}},
	{"/new", "开始新会话", nil},
	{"/cancel", "取消当前运行中的会话", nil},
	{"/clear", "清除屏幕内容", nil},
	{"/exit", "退出程序", nil},
}

var (
	sCmdName = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	sCmdDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sCmdSel  = lipgloss.NewStyle().Background(lipgloss.Color("237"))
)

func filterCommands(prefix string) []slashCmd {
	if prefix == "/" {
		return slashCommands
	}
	p := strings.ToLower(prefix)
	var out []slashCmd
	for _, c := range slashCommands {
		if strings.HasPrefix(c.Name, p) {
			out = append(out, c)
		}
	}
	return out
}

func findCmdSubOpts(cmdName string) []slashCmd {
	for _, c := range slashCommands {
		if c.Name == cmdName {
			return c.SubOpts
		}
	}
	return nil
}

func filterSubOpts(opts []slashCmd, prefix string) []slashCmd {
	if prefix == "" {
		return opts
	}
	p := strings.ToLower(prefix)
	var out []slashCmd
	for _, o := range opts {
		if strings.HasPrefix(strings.ToLower(o.Name), p) {
			out = append(out, o)
		}
	}
	return out
}

func buildProviderCmdOptions(cur *SettingResponse, providers []ProviderOption) []slashCmd {
	currentProviderID := int64(0)
	if cur != nil {
		currentProviderID = cur.SelectedProviderID
	}

	result := make([]slashCmd, 0, len(providers))
	for _, item := range providers {
		name := strings.TrimSpace(item.Type)
		if item.ID > 0 {
			name = strconv.FormatInt(item.ID, 10)
		}
		if name == "" {
			continue
		}

		descParts := make([]string, 0, 4)
		if item.Type != "" {
			descParts = append(descParts, item.Type)
		}
		if item.Domain != "" {
			descParts = append(descParts, item.Domain)
		}
		if item.Disabled {
			descParts = append(descParts, "disabled")
		}
		if currentProviderID > 0 && item.ID == currentProviderID {
			descParts = append(descParts, "当前")
		}

		desc := strings.Join(descParts, " · ")
		if desc == "" {
			desc = "provider"
		}
		result = append(result, slashCmd{Name: name, Desc: desc})
	}
	return result
}

func resolveProviderConfig(cur *SettingResponse, providers []ProviderOption) string {
	if cur == nil {
		return ""
	}
	if cur.SelectedProviderID > 0 {
		for _, item := range providers {
			if item.ID == cur.SelectedProviderID {
				if providerType := strings.TrimSpace(item.Type); providerType != "" {
					return providerType
				}
				break
			}
		}
	}
	return strings.TrimSpace(cur.AIService)
}

func buildModelCmdOptions(cur *SettingResponse, providerType string, models []string) []slashCmd {
	currentModel := ""
	if cur != nil {
		currentModel = strings.TrimSpace(cur.SelectedModelName)
		if currentModel == "" {
			currentModel = strings.TrimSpace(cur.AIModelName)
		}
	}

	result := make([]slashCmd, 0, len(models))
	for _, item := range models {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		descParts := make([]string, 0, 2)
		if providerType != "" {
			descParts = append(descParts, providerType)
		}
		if currentModel != "" && strings.EqualFold(currentModel, name) {
			descParts = append(descParts, "当前")
		}
		desc := strings.Join(descParts, " · ")
		if desc == "" {
			desc = "model"
		}
		result = append(result, slashCmd{Name: name, Desc: desc})
	}
	return result
}

// ---------- stream metadata ----------

type streamMeta struct {
	nodeID      string
	contentType string
}

var nodeVerboseMap = map[string]string{
	"re-act-loop":                          "推理与行动",
	"re-act-loop-thought":                  "思考",
	"re-act-loop-answer-payload":           "AI 响应",
	"call-forge":                           "智能应用",
	"call-tools":                           "工具调用",
	"generating-tool-call-params":          "生成工具参数",
	"review":                               "审查系统",
	"liteforge":                            "轻量智能应用",
	"directly_answer":                      "直接回答",
	"directly-answer":                      "直接回答",
	"memory-reducer":                       "记忆裁剪",
	"memory-timeline":                      "记忆浓缩",
	"execute":                              "执行",
	"summary":                              "总结",
	"summary-status":                       "状态总结",
	"summary-short":                        "简短总结",
	"summary-long":                         "详细总结",
	"create-subtasks":                      "创建子任务",
	"freedom-plan-review":                  "计划审查",
	"dynamic-plan":                         "动态规划",
	"re-act-verify":                        "核实结果",
	"result":                               "结果输出",
	"human_readable_result":                "结果摘要",
	"plan":                                 "任务规划",
	"decision":                             "决策",
	"output":                               "通用输出",
	"forge":                                "智能应用",
	"enhance-query":                        "知识增强",
	"knowledge_enhance":                    "知识增强",
	"knowledge":                            "知识",
	"internet-research":                    "互联网调研",
	"intent":                               "意图识别",
	"report_generating":                    "报告生成",
	"report-content":                       "报告内容",
	"write_code":                           "编写代码",
	"write_yaklang_code":                   "编写 Yaklang 代码",
	"modify_code":                          "修改代码",
	"gen_code":                             "生成代码",
	"verification":                         "验证",
	"answer":                               "回答",
	"action":                               "行动",
	"self-reflection":                      "自我反思",
	"self-reflection-suggestions":          "自省建议",
	"self-reflection-learning_insights":    "学习洞察",
	"self-reflection-future_suggestions":   "未来建议",
	"self-reflection-impact_assessment":    "影响评估",
	"self-reflection-effectiveness_rating": "效果评级",
	"interval-review":                      "进度检查",
	"task-analyst":                         "任务分析",
	"focus-on":                             "专注",
	"knowledge-compress":                   "知识压缩",
	"batch-compress":                       "记忆压缩",
	"reference_material":                   "参考资料",
	"http_flow_analyze":                    "HTTP 流量分析",
}

// 主要输出 nodeID：直接显示文本，不带节标题
var primaryOutputNodes = map[string]bool{
	"re-act-loop-answer-payload": true,
	"directly_answer":            true,
	"directly-answer":            true,
	"output":                     true,
	"human_readable_result":      true,
	"answer":                     true,
	"report-content":             true,
}

// 静默 nodeID：内部状态/系统事件，不显示
var silentNodes = map[string]bool{
	"status":                            true,
	"queue_info":                        true,
	"timeline":                          true,
	"timeline_item":                     true,
	"react_task_enqueue":                true,
	"react_task_dequeue":                true,
	"react_task_cleared":                true,
	"react_task_status_changed":         true,
	"react_task_mode_changed":           true,
	"react_task_created":                true,
	"plan-executing-loading-status-key": true,
	"stream-finished":                   true,
	"system":                            true,
	"prompt":                            true,
	"base":                              true,
	"debug":                             true,
	"trace":                             true,
	"load_capability":                   true,
	"loading_skills_name":               true,
	"lose-focus":                        true,
}

func verboseLabel(nodeID string) string {
	if v, ok := nodeVerboseMap[nodeID]; ok {
		return v
	}
	return nodeID
}

func isToolStdout(nodeID string) bool {
	return strings.HasPrefix(nodeID, "tool-") && strings.HasSuffix(nodeID, "-stdout")
}

func isToolStderr(nodeID string) bool {
	return strings.HasPrefix(nodeID, "tool-") && strings.HasSuffix(nodeID, "-stderr")
}

func shouldHideStream(nodeID, contentType string) bool {
	if isToolStderr(nodeID) {
		return true
	}
	if contentType == "log/tool-error-output" {
		return true
	}
	if silentNodes[nodeID] {
		return true
	}
	return false
}

func isPrimaryOutput(nodeID string) bool {
	return primaryOutputNodes[nodeID]
}

func streamSectionHeader(nodeID, contentType string) string {
	if isPrimaryOutput(nodeID) || nodeID == "" {
		return ""
	}
	switch {
	case nodeID == "re-act-loop-thought" || nodeID == "thought":
		return sThought.Render("💭 思考")
	case nodeID == "plan" || nodeID == "dynamic-plan":
		return sLabel.Render("📋 " + verboseLabel(nodeID))
	case nodeID == "decision":
		return sLabel.Render("🎯 决策")
	case nodeID == "execute":
		return sLabel.Render("⚡ 执行")
	case nodeID == "re-act-verify":
		return sLabel.Render("✅ 核实结果")
	case strings.HasPrefix(nodeID, "summary"):
		return sLabel.Render("📝 " + verboseLabel(nodeID))
	case nodeID == "call-tools" || nodeID == "generating-tool-call-params":
		return sTool.Render("🔧 " + verboseLabel(nodeID))
	case isToolStdout(nodeID):
		toolName := strings.TrimSuffix(strings.TrimPrefix(nodeID, "tool-"), "-stdout")
		return sTool.Render("📄 " + toolName)
	case nodeID == "call-forge" || nodeID == "forge" || nodeID == "liteforge":
		return sLabel.Render("🤖 " + verboseLabel(nodeID))
	case nodeID == "review" || nodeID == "interval-review" || nodeID == "freedom-plan-review":
		return sLabel.Render("👁 " + verboseLabel(nodeID))
	case nodeID == "enhance-query" || nodeID == "knowledge_enhance" || nodeID == "knowledge":
		return sLabel.Render("📚 " + verboseLabel(nodeID))
	case strings.HasPrefix(nodeID, "self-reflection"):
		return sLabel.Render("🔍 " + verboseLabel(nodeID))
	case contentType == "code/yaklang" || contentType == "code/http-request":
		label := verboseLabel(nodeID)
		if label == nodeID {
			label = "代码"
		}
		return sLabel.Render("💻 " + label)
	default:
		label := verboseLabel(nodeID)
		if label != nodeID {
			return sLabel.Render("[" + label + "]")
		}
		return ""
	}
}

// ---------- TUIModel ----------

type TUIModel struct {
	client    *AgentClient
	serverURL string

	input    textinput.Model
	viewport viewport.Model
	spin     spinner.Model

	phase             tuiPhase
	content           string
	streamBuf         string
	streamNodeID      string
	streamContentType string
	currentRunID      string
	pendingQuery      string
	statusText        string

	activeStreams map[string]*streamMeta

	eventCh   <-chan SSEEvent
	sseCancel context.CancelFunc

	interType  string
	interEvent *RunEvent

	aiService string
	aiModel   string

	providerCmdOpts     []slashCmd
	modelCmdOpts        []slashCmd
	loadingProviderOpts bool
	loadingModelOpts    bool

	cmdMatches []slashCmd
	cmdIdx     int

	w, h  int
	ready bool
}

func NewTUIModel(client *AgentClient, serverURL string) TUIModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "输入消息或 /help 查看命令..."
	ti.Focus()
	ti.CharLimit = 0
	inputBg := lipgloss.Color("235")
	ti.PromptStyle = lipgloss.NewStyle().
		Background(inputBg).
		Foreground(lipgloss.Color("205"))
	ti.TextStyle = lipgloss.NewStyle().
		Background(inputBg).
		Foreground(lipgloss.Color("252"))
	ti.PlaceholderStyle = lipgloss.NewStyle().
		Background(inputBg).
		Foreground(lipgloss.Color("240"))
	ti.CompletionStyle = lipgloss.NewStyle().
		Background(inputBg).
		Foreground(lipgloss.Color("240"))
	ti.Cursor.TextStyle = ti.PlaceholderStyle
	ti.Cursor.Style = lipgloss.NewStyle().
		Background(inputBg).
		Foreground(lipgloss.Color("252"))

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return TUIModel{
		client:        client,
		serverURL:     serverURL,
		input:         ti,
		spin:          sp,
		phase:         phaseConnecting,
		activeStreams: make(map[string]*streamMeta),
	}
}

// ---------- Init / Update / View ----------

func (m TUIModel) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.cmdConnect())
}

func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.relayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		var c tea.Cmd
		m.viewport, c = m.viewport.Update(msg)
		return m, c

	case spinner.TickMsg:
		var c tea.Cmd
		m.spin, c = m.spin.Update(msg)
		if m.phase == phaseConnecting || m.phase == phaseCreating || m.phase == phaseStreaming {
			m.refreshVP()
		}
		return m, c

	case connectResultMsg:
		return m.onConnect(msg), nil

	case sessionCreatedMsg:
		return m.onSessionCreated(msg)

	case sseStartedMsg:
		m.eventCh = msg.ch
		m.sseCancel = msg.cancel
		m.phase = phaseStreaming
		return m, listenSSE(m.eventCh)

	case sseEventMsg:
		extra := m.onSSEEvent(SSEEvent(msg))
		m.refreshVP()
		if extra != nil {
			cmds = append(cmds, extra)
		}
		if m.eventCh != nil {
			cmds = append(cmds, listenSSE(m.eventCh))
		}
		return m, tea.Batch(cmds...)

	case sseDoneMsg:
		m.flushStream()
		m.phase = phaseReady
		m.statusText = ""
		m.eventCh = nil
		m.refreshVP()
		return m, nil

	case sseErrorMsg:
		m.phase = phaseReady
		m.appendErr(fmt.Sprintf("SSE 错误: %v", msg.err))
		m.currentRunID = ""
		m.refreshVP()
		return m, nil

	case pushResultMsg:
		if msg.err != nil {
			m.appendErr(fmt.Sprintf("发送失败: %v", msg.err))
			m.refreshVP()
		}
		return m, nil

	case cancelResultMsg:
		if msg.err != nil {
			m.appendErr(fmt.Sprintf("取消失败: %v", msg.err))
		} else {
			m.appendSys("已取消")
		}
		m.refreshVP()
		return m, nil

	case providerOptionsLoadedMsg:
		m.loadingProviderOpts = false
		if msg.err != nil {
			m.appendErr(fmt.Sprintf("加载 provider 选项失败: %v", msg.err))
		} else {
			m.providerCmdOpts = msg.options
		}
		m.refreshCmdPalette()
		m.refreshVP()
		return m, nil

	case modelOptionsLoadedMsg:
		m.loadingModelOpts = false
		if msg.err != nil {
			m.appendErr(fmt.Sprintf("加载 model 选项失败: %v", msg.err))
		} else {
			m.modelCmdOpts = msg.options
		}
		m.refreshCmdPalette()
		m.refreshVP()
		return m, nil
	}

	var c tea.Cmd
	m.input, c = m.input.Update(msg)
	if c != nil {
		cmds = append(cmds, c)
	}
	return m, tea.Batch(cmds...)
}

func (m TUIModel) renderCmdPalette() string {
	if len(m.cmdMatches) == 0 {
		return ""
	}
	var b strings.Builder
	for i, c := range m.cmdMatches {
		name := sCmdName.Render(fmt.Sprintf("  %-16s", c.Name))
		desc := sCmdDesc.Render(c.Desc)
		line := name + desc
		if i == m.cmdIdx {
			line = sCmdSel.Width(m.w).Render(line)
		}
		b.WriteString(line)
		if i < len(m.cmdMatches)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m TUIModel) View() string {
	if !m.ready {
		return m.spin.View() + " 初始化中..."
	}

	var b strings.Builder
	b.WriteString(sHeaderBar.Width(m.w).Render(m.headerLine()))
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	if palette := m.renderCmdPalette(); palette != "" {
		b.WriteString(palette)
		b.WriteString("\n")
	}

	inputContent := m.input.View()
	b.WriteString(sInputBar.Width(m.w - 4).Render(inputContent))
	b.WriteString("\n")

	return b.String()
}

// ---------- key handling ----------

func (m *TUIModel) refreshCmdPalette() {
	prev := len(m.cmdMatches)
	val := m.input.Value()

	m.cmdMatches = nil
	m.cmdIdx = 0

	if strings.HasPrefix(val, "/") {
		spIdx := strings.Index(val, " ")
		if spIdx < 0 {
			m.cmdMatches = filterCommands(val)
		} else {
			cmdName := val[:spIdx]
			argPrefix := val[spIdx+1:]
			if !strings.Contains(argPrefix, " ") {
				if subs := m.findCmdSubOpts(cmdName); len(subs) > 0 {
					m.cmdMatches = filterSubOpts(subs, argPrefix)
				}
			}
		}
		if m.cmdIdx >= len(m.cmdMatches) {
			m.cmdIdx = 0
		}
	}
	if len(m.cmdMatches) != prev {
		m.relayout()
	}
}

func (m *TUIModel) findCmdSubOpts(cmdName string) []slashCmd {
	switch cmdName {
	case "/provider":
		return m.providerCmdOpts
	case "/model":
		return m.modelCmdOpts
	default:
		return findCmdSubOpts(cmdName)
	}
}

func (m *TUIModel) paletteIsSubMode() bool {
	val := m.input.Value()
	return strings.HasPrefix(val, "/") && strings.Contains(val, " ")
}

func (m *TUIModel) invalidateDynamicCmdOptions() {
	m.providerCmdOpts = nil
	m.modelCmdOpts = nil
	m.loadingProviderOpts = false
	m.loadingModelOpts = false
}

func (m *TUIModel) cmdLoadProviderOptions() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		setting, err := c.GetSetting()
		if err != nil {
			return providerOptionsLoadedMsg{err: err}
		}

		providers, err := c.ListAIProviders()
		if err != nil {
			return providerOptionsLoadedMsg{err: err}
		}
		return providerOptionsLoadedMsg{
			options: buildProviderCmdOptions(setting, providers),
		}
	}
}

func (m *TUIModel) cmdLoadModelOptions() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		setting, err := c.GetSetting()
		if err != nil {
			return modelOptionsLoadedMsg{err: err}
		}

		providers, err := c.ListAIProviders()
		if err != nil {
			return modelOptionsLoadedMsg{err: err}
		}
		providerType := resolveProviderConfig(setting, providers)
		if providerType == "" {
			return modelOptionsLoadedMsg{err: fmt.Errorf("当前未选择 provider，请先执行 /provider")}
		}

		models, err := c.ListAIModels(providerType)
		if err != nil {
			return modelOptionsLoadedMsg{err: err}
		}
		return modelOptionsLoadedMsg{
			options: buildModelCmdOptions(setting, providerType, models),
		}
	}
}

func (m *TUIModel) maybeLoadDynamicCmdOptions() tea.Cmd {
	val := m.input.Value()

	if strings.HasPrefix(val, "/provider ") {
		argPrefix := strings.TrimPrefix(val, "/provider ")
		if !strings.Contains(argPrefix, " ") && !m.loadingProviderOpts && len(m.providerCmdOpts) == 0 {
			m.loadingProviderOpts = true
			return m.cmdLoadProviderOptions()
		}
	}

	if strings.HasPrefix(val, "/model ") {
		argPrefix := strings.TrimPrefix(val, "/model ")
		if !strings.Contains(argPrefix, " ") && !m.loadingModelOpts && len(m.modelCmdOpts) == 0 {
			m.loadingModelOpts = true
			return m.cmdLoadModelOptions()
		}
	}

	return nil
}

func (m TUIModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	paletteOpen := len(m.cmdMatches) > 0

	switch msg.Type {
	case tea.KeyCtrlC:
		if m.phase == phaseStreaming && m.currentRunID != "" {
			runID := m.currentRunID
			if m.sseCancel != nil {
				m.sseCancel()
			}
			m.phase = phaseReady
			m.currentRunID = ""
			m.statusText = ""
			m.refreshVP()
			return m, m.cmdCancelRun(runID)
		}
		return m, tea.Quit

	case tea.KeyEsc:
		if paletteOpen {
			m.cmdMatches = nil
			m.cmdIdx = 0
			m.relayout()
			return m, nil
		}

	case tea.KeyTab:
		if paletteOpen {
			sel := m.cmdMatches[m.cmdIdx]
			var newVal string
			if m.paletteIsSubMode() {
				cmdPart := m.input.Value()[:strings.Index(m.input.Value(), " ")+1]
				newVal = cmdPart + sel.Name + " "
			} else {
				newVal = sel.Name + " "
			}
			m.input.SetValue(newVal)
			m.input.SetCursor(len(newVal))
			m.cmdMatches = nil
			m.cmdIdx = 0
			m.refreshCmdPalette()
			return m, nil
		}

	case tea.KeyEnter:
		if paletteOpen {
			sel := m.cmdMatches[m.cmdIdx]
			if m.paletteIsSubMode() {
				cmdPart := m.input.Value()[:strings.Index(m.input.Value(), " ")+1]
				newVal := cmdPart + sel.Name
				m.input.SetValue(newVal)
				m.input.SetCursor(len(newVal))
			} else {
				if findCmdSubOpts(sel.Name) != nil {
					newVal := sel.Name + " "
					m.input.SetValue(newVal)
					m.input.SetCursor(len(newVal))
					m.cmdMatches = nil
					m.cmdIdx = 0
					m.refreshCmdPalette()
					return m, nil
				}
				m.input.SetValue(sel.Name)
				m.input.SetCursor(len(sel.Name))
			}
			m.cmdMatches = nil
			m.cmdIdx = 0
			m.relayout()
			return m.handleSubmit()
		}
		return m.handleSubmit()

	case tea.KeyUp:
		if paletteOpen {
			if m.cmdIdx > 0 {
				m.cmdIdx--
			} else {
				m.cmdIdx = len(m.cmdMatches) - 1
			}
			return m, nil
		}
		var c tea.Cmd
		m.viewport, c = m.viewport.Update(msg)
		return m, c

	case tea.KeyDown:
		if paletteOpen {
			if m.cmdIdx < len(m.cmdMatches)-1 {
				m.cmdIdx++
			} else {
				m.cmdIdx = 0
			}
			return m, nil
		}
		var c tea.Cmd
		m.viewport, c = m.viewport.Update(msg)
		return m, c

	case tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		var c tea.Cmd
		m.viewport, c = m.viewport.Update(msg)
		return m, c
	}

	var c tea.Cmd
	m.input, c = m.input.Update(msg)
	m.refreshCmdPalette()
	loadCmd := m.maybeLoadDynamicCmdOptions()
	if c == nil {
		return m, loadCmd
	}
	if loadCmd == nil {
		return m, c
	}
	return m, tea.Batch(c, loadCmd)
}

func (m TUIModel) handleSubmit() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return m, nil
	}
	m.input.SetValue("")

	if m.phase == phaseInteractive {
		return m.respondInteractive(text)
	}
	if strings.HasPrefix(text, "/") {
		return m.handleCmd(text)
	}
	if m.phase != phaseReady && m.phase != phaseStreaming {
		return m, nil
	}

	m.appendUser(text)

	if m.currentRunID != "" {
		if m.eventCh == nil {
			if m.phase == phaseStreaming {
				m.appendSys("会话启动中，请稍候…")
			} else {
				m.appendErr("事件流未连接，请输入 /new 重新创建会话")
			}
			m.refreshVP()
			return m, nil
		}
		m.phase = phaseStreaming
		m.refreshVP()
		return m, m.cmdPush(PushEventRequest{Type: "free_input", FreeInput: text})
	}

	m.pendingQuery = text
	m.phase = phaseCreating
	m.refreshVP()
	return m, m.cmdCreateSession()
}

// ---------- commands (slash) ----------

func (m TUIModel) handleCmd(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	switch strings.ToLower(parts[0]) {
	case "/help":
		m.appendSys(`可用命令:
  /help              显示帮助
  /setting           查看当前设置
  /provider          查看/选择 provider
  /model             查看/选择 model
  /focus_mode        查看/选择 focus mode
  /set <key> <v>     更新设置 (service / model / forge / policy / provider / selected_model / focus_mode)
  /review_policy <p> 设置审查策略 (auto / ai / manual / ai-auto)
  /new               开始新会话
  /cancel            取消当前会话
  /clear             清除屏幕
  /exit, /quit       退出`)

	case "/exit", "/quit":
		return m, tea.Quit

	case "/clear":
		m.content = ""

	case "/setting":
		s, err := m.client.GetSetting()
		if err != nil {
			m.appendErr(err.Error())
		} else {
			data, _ := json.MarshalIndent(s, "", "  ")
			m.appendSys("当前设置:\n" + string(data))
		}

	case "/set":
		if len(parts) < 3 {
			m.appendSys("用法: /set <key> <value>")
		} else {
			m.applySetting(parts[1], strings.Join(parts[2:], " "))
		}

	case "/provider":
		if len(parts) == 1 {
			m.input.SetValue("/provider ")
			m.input.SetCursor(len("/provider "))
			m.refreshCmdPalette()
			if !m.loadingProviderOpts {
				m.loadingProviderOpts = true
				m.refreshVP()
				return m, m.cmdLoadProviderOptions()
			}
			m.refreshVP()
			return m, nil
		}
		m.handleProviderCommand(parts[1:])

	case "/model":
		if len(parts) == 1 {
			m.input.SetValue("/model ")
			m.input.SetCursor(len("/model "))
			m.refreshCmdPalette()
			if !m.loadingModelOpts {
				m.loadingModelOpts = true
				m.refreshVP()
				return m, m.cmdLoadModelOptions()
			}
			m.refreshVP()
			return m, nil
		}
		m.handleModelCommand(parts[1:])

	case "/focus_mode":
		m.handleFocusModeCommand(parts[1:])

	case "/review_policy":
		if len(parts) < 2 {
			m.appendSys("用法: /review_policy <auto|ai|manual|ai-auto>")
		} else {
			m.applySetting("policy", parts[1])
		}

	case "/new":
		if m.sseCancel != nil {
			m.sseCancel()
		}
		m.currentRunID = ""
		m.statusText = ""
		m.phase = phaseReady
		m.appendSys("已准备创建新会话，输入消息即可开始")

	case "/cancel":
		if m.currentRunID != "" {
			runID := m.currentRunID
			if m.sseCancel != nil {
				m.sseCancel()
			}
			m.phase = phaseReady
			m.currentRunID = ""
			m.statusText = ""
			m.refreshVP()
			return m, m.cmdCancelRun(runID)
		} else {
			m.appendSys("没有运行中的会话")
		}

	default:
		m.appendSys(fmt.Sprintf("未知命令: %s (输入 /help 查看帮助)", parts[0]))
	}
	m.refreshVP()
	return m, nil
}

// ---------- tea.Cmd factories ----------

func (m TUIModel) cmdConnect() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		s, err := c.GetSetting()
		return connectResultMsg{setting: s, err: err}
	}
}

func (m TUIModel) cmdCreateSession() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		resp, err := c.CreateSession("", GenerateTaskID(), AIParams{UseDefaultAI: true})
		if err != nil {
			return sessionCreatedMsg{err: err}
		}
		return sessionCreatedMsg{runID: resp.RunID}
	}
}

func (m TUIModel) cmdRunAndStream(query string) tea.Cmd {
	c, rid := m.client, m.currentRunID
	return func() tea.Msg {
		ch := make(chan SSEEvent, 256)
		ctx, cancel := context.WithCancel(context.Background())
		req := PushEventRequest{Type: "free_input", FreeInput: query}
		if err := c.RunAndStream(ctx, rid, req, ch); err != nil {
			cancel()
			return sseErrorMsg{err: err}
		}
		return sseStartedMsg{ch: ch, cancel: cancel}
	}
}

func listenSSE(ch <-chan SSEEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return sseDoneMsg{}
		}
		return sseEventMsg(ev)
	}
}

func (m TUIModel) cmdPush(req PushEventRequest) tea.Cmd {
	c, rid := m.client, m.currentRunID
	return func() tea.Msg {
		return pushResultMsg{err: c.PushEvent(rid, req)}
	}
}

func (m TUIModel) cmdCancelRun(runID string) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		return cancelResultMsg{err: c.CancelRun(runID)}
	}
}

// ---------- message handlers ----------

func (m *TUIModel) onConnect(msg connectResultMsg) TUIModel {
	if msg.err != nil {
		m.appendErr(fmt.Sprintf("连接失败: %v", msg.err))
		m.phase = phaseReady
	} else {
		m.phase = phaseReady
		m.aiService = msg.setting.AIService
		m.aiModel = msg.setting.AIModelName
		m.invalidateDynamicCmdOptions()
		m.loadingProviderOpts = false
		m.loadingModelOpts = false
		m.appendSys("✓ 已连接到 " + m.serverURL)
		if m.aiService != "" {
			m.appendSys(fmt.Sprintf("  AI: %s / %s", m.aiService, m.aiModel))
		}
		m.appendSys("输入 /help 查看可用命令")
	}
	m.refreshVP()
	return *m
}

func (m TUIModel) onSessionCreated(msg sessionCreatedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.phase = phaseReady
		m.appendErr(fmt.Sprintf("创建会话失败: %v", msg.err))
		m.refreshVP()
		return m, nil
	}
	m.currentRunID = msg.runID
	m.appendSys(fmt.Sprintf("会话: %s", shortID(msg.runID)))
	m.refreshVP()
	q := m.pendingQuery
	m.pendingQuery = ""
	return m, m.cmdRunAndStream(q)
}

func (m *TUIModel) onSSEEvent(ev SSEEvent) tea.Cmd {
	var e RunEvent
	if err := json.Unmarshal([]byte(ev.Data), &e); err != nil {
		return nil
	}

	typ := ev.Event
	if e.Type != "" {
		typ = e.Type
	}

	switch typ {
	case "listener_ready":
		// run request is triggered by RunAndStream after listener_ready.

	case "stream_start":
		if e.EventUUID != "" {
			m.activeStreams[e.EventUUID] = &streamMeta{
				nodeID:      e.NodeID,
				contentType: e.Content,
			}
		}

	case "stream":
		m.handleStreamDelta(e)

	case "structured":
		if e.NodeID == "stream-finished" {
			writerID := extractField(e.Content, "event_writer_id")
			if writerID != "" {
				delete(m.activeStreams, writerID)
			}
			m.flushStream()
		} else if e.NodeID == "status" {
			v := extractField(e.Content, "value")
			if v == "end" {
				m.flushStream()
				m.statusText = ""
				m.phase = phaseReady
			} else if v != "" {
				m.statusText = v
			}
		}

	case "thought":
		m.flushStream()
		t := simpleText(e.Content, "thought")
		if t != "" {
			m.appendBlock(sThought.Render("💭 " + t))
		}

	case "tool_call_start":
		m.flushStream()
		name := extractField(e.Content, "tool_name")
		if name == "" {
			name = extractField(e.Content, "name")
		}
		if name == "" {
			name = "tool"
		}
		m.appendBlock(sTool.Render("🔧 " + e.Content))

	case "tool_call_done":
		m.flushStream()
		name := extractField(e.Content, "tool_name")
		if name == "" {
			name = "tool"
		}
		m.appendBlock(sTool.Render("   ✓ " + name))

	case "tool_call_error":
		m.flushStream()
		name := extractField(e.Content, "tool_name")
		if name == "" {
			name = "tool"
		}
		m.appendBlock(sTool.Render("   ✗ " + name))

	case "require_user_interactive",
		"plan_review_require", "task_review_require",
		"tool_use_review_require", "exec_aiforge_review_require",
		"permission_require":
		m.flushStream()
		m.phase = phaseInteractive
		m.interType = typ
		m.interEvent = &e
		m.showInterPrompt(typ, e)
		m.input.Placeholder = "输入回复 (y/n 或自由输入)..."

	case "done":
		m.flushStream()
		m.statusText = ""
		m.appendBlock(sSystem.Render("━━━ 回复完成 ━━━"))

	case "error":
		m.flushStream()
		m.statusText = ""
		m.appendErr("会话失败: " + simpleText(e.Content, "error"))

	default:
		// silently ignore unrecognized / system events
	}
	return nil
}

var allowStreamNodes = map[string]bool{
	"re-act-loop-thought":        true,
	"re-act-loop-answer-payload": true,
}

func (m *TUIModel) handleStreamDelta(e RunEvent) {
	nodeID := e.NodeID
	if !allowStreamNodes[nodeID] {
		return
	}
	contentType := ""

	if e.EventUUID != "" {
		if meta, ok := m.activeStreams[e.EventUUID]; ok {
			if nodeID == "" {
				nodeID = meta.nodeID
			}
			contentType = meta.contentType
		}
	}

	if shouldHideStream(nodeID, contentType) {
		return
	}

	delta := e.StreamDelta
	if delta == "" {
		delta = e.Content
	}
	if delta == "" {
		return
	}

	if nodeID != m.streamNodeID {
		m.flushStream()
		m.streamNodeID = nodeID
		m.streamContentType = contentType
	}

	m.streamBuf += delta
}

func (m TUIModel) respondInteractive(text string) (tea.Model, tea.Cmd) {
	if m.interEvent == nil || m.currentRunID == "" {
		m.phase = phaseStreaming
		return m, nil
	}
	m.appendUser(text)

	var req PushEventRequest
	if m.interType == "require_user_interactive" {
		req = PushEventRequest{Type: "free_input", FreeInput: text}
	} else {
		approved := strings.EqualFold(text, "y") || strings.EqualFold(text, "yes")
		resp, _ := json.Marshal(map[string]any{"approved": approved, "message": text})
		req = PushEventRequest{
			Type:                 "interactive",
			InteractiveID:        m.interEvent.EventUUID,
			InteractiveJSONInput: string(resp),
		}
	}

	m.phase = phaseStreaming
	m.interType = ""
	m.interEvent = nil
	m.input.Placeholder = "输入消息或 /help 查看命令..."
	m.refreshVP()
	return m, m.cmdPush(req)
}

// ---------- content helpers ----------

func (m *TUIModel) appendBlock(block string) {
	if strings.TrimSpace(block) == "" {
		return
	}
	m.content += block + "\n\n"
}

func (m *TUIModel) appendUser(t string) {
	width := m.w - 2
	if width < 1 {
		width = 1
	}
	m.appendBlock(sUserBar.Width(width).Render(t))
}

func (m *TUIModel) appendSys(t string) { m.appendBlock(sSystem.Render(t)) }
func (m *TUIModel) appendErr(t string) { m.appendBlock(sErr.Render("✗ " + t)) }

func (m *TUIModel) flushStream() {
	if m.streamBuf == "" {
		return
	}
	text := m.streamBuf
	nodeID := m.streamNodeID
	ct := m.streamContentType
	m.streamBuf = ""
	m.streamNodeID = ""
	m.streamContentType = ""

	switch {
	case nodeID == "re-act-loop-thought" || nodeID == "thought":
		m.appendBlock(sDim.Render(text))

	case isToolStdout(nodeID):
		m.appendBlock(sDim.Render(text))

	case ct == "code/yaklang" || ct == "code/http-request":
		m.appendBlock(sDim.Render("```") + "\n" + text + "\n" + sDim.Render("```"))

	case ct == "log/tool":
		m.appendBlock(sDim.Render(text))

	default:
		if isPrimaryOutput(nodeID) {
			m.appendAIBlock(text)
		} else {
			m.appendBlock(text)
		}
	}
}

func (m *TUIModel) appendAIBlock(text string) {
	content := text
	modelName := strings.TrimSpace(m.aiModel)
	if modelName != "" {
		content += "\n" + sModel.Render("◧ "+modelName)
	}
	m.appendBlock(content)
}

func (m *TUIModel) showInterPrompt(typ string, e RunEvent) {
	labels := map[string]string{
		"require_user_interactive":    "💬 AI 请求输入:",
		"plan_review_require":         "📋 计划审核:",
		"task_review_require":         "📝 任务审核:",
		"tool_use_review_require":     "🔧 工具使用审核:",
		"exec_aiforge_review_require": "🤖 Forge 审核:",
		"permission_require":          "🔐 权限请求:",
	}
	label := labels[typ]
	if label == "" {
		label = typ
	}
	block := sPrompt.Render(label)

	if e.Content != "" {
		var data map[string]any
		if json.Unmarshal([]byte(e.Content), &data) == nil {
			if msg, ok := data["message"].(string); ok {
				block += "\n  " + msg
			} else if p, ok := data["prompt"].(string); ok {
				block += "\n  " + p
			}
		} else {
			block += "\n  " + e.Content
		}
	}
	m.appendBlock(block)
}

func buildSettingRequest(cur *SettingResponse) SettingRequest {
	return SettingRequest{
		UseDefaultAIConfig: cur.UseDefaultAIConfig,
		AIService:          cur.AIService,
		AIModelName:        cur.AIModelName,
		ForgeName:          cur.ForgeName,
		ReviewPolicy:       cur.ReviewPolicy,
		MaxIteration:       cur.MaxIteration,
		DisableToolUse:     cur.DisableToolUse,
		SelectedProviderID: cur.SelectedProviderID,
		SelectedModelName:  cur.SelectedModelName,
		SelectedModelTier:  cur.SelectedModelTier,
		FocusModeLoop:      cur.FocusModeLoop,
	}
}

func (m *TUIModel) handleProviderCommand(args []string) {
	cur, err := m.client.GetSetting()
	if err != nil {
		m.appendErr(fmt.Sprintf("读取设置失败: %v", err))
		return
	}

	providers, err := m.client.ListAIProviders()
	if err != nil {
		m.appendErr(fmt.Sprintf("获取 provider 列表失败: %v", err))
		return
	}
	if len(providers) == 0 {
		m.appendSys("未获取到可用 provider")
		return
	}

	if len(args) == 0 {
		var b strings.Builder
		b.WriteString("可用 provider（用法: /provider <id|type>）:")
		for _, item := range providers {
			prefix := "  "
			if item.ID == cur.SelectedProviderID {
				prefix = "* "
			}
			line := fmt.Sprintf("%s%d  %s", prefix, item.ID, item.Type)
			if item.Domain != "" {
				line += "  (" + item.Domain + ")"
			}
			if item.Disabled {
				line += " [disabled]"
			}
			b.WriteString("\n" + line)
		}
		m.appendSys(b.String())
		return
	}

	targetArg := strings.TrimSpace(strings.Join(args, " "))
	var selected *ProviderOption
	for idx := range providers {
		item := &providers[idx]
		if strconv.FormatInt(item.ID, 10) == targetArg || strings.EqualFold(item.Type, targetArg) {
			selected = item
			break
		}
	}
	if selected == nil {
		m.appendErr(fmt.Sprintf("未找到 provider: %s", targetArg))
		return
	}

	req := buildSettingRequest(cur)
	req.SelectedProviderID = selected.ID
	req.AIService = selected.Type

	resp, err := m.client.UpdateSetting(req)
	if err != nil {
		m.appendErr(fmt.Sprintf("更新 provider 失败: %v", err))
		return
	}
	m.invalidateDynamicCmdOptions()
	m.aiService = resp.AIService
	m.aiModel = resp.AIModelName
	m.appendSys(fmt.Sprintf("✓ Provider 已更新: %d / %s", selected.ID, selected.Type))
}

func (m *TUIModel) handleModelCommand(args []string) {
	cur, err := m.client.GetSetting()
	if err != nil {
		m.appendErr(fmt.Sprintf("读取设置失败: %v", err))
		return
	}

	providers, err := m.client.ListAIProviders()
	if err != nil {
		m.appendErr(fmt.Sprintf("获取 provider 列表失败: %v", err))
		return
	}
	config := resolveProviderConfig(cur, providers)
	if config == "" {
		m.appendErr("当前未选择 provider，请先执行 /provider")
		return
	}

	models, err := m.client.ListAIModels(config)
	if err != nil {
		m.appendErr(fmt.Sprintf("获取 model 列表失败: %v", err))
		return
	}
	if len(models) == 0 {
		m.appendSys(fmt.Sprintf("provider %s 暂无可选 model", config))
		return
	}

	currentModel := strings.TrimSpace(cur.SelectedModelName)
	if currentModel == "" {
		currentModel = strings.TrimSpace(cur.AIModelName)
	}

	if len(args) == 0 {
		var b strings.Builder
		b.WriteString(fmt.Sprintf("provider=%s 的可用 model（用法: /model <名称|序号>）:", config))
		for idx, name := range models {
			prefix := "  "
			if strings.EqualFold(name, currentModel) {
				prefix = "* "
			}
			b.WriteString(fmt.Sprintf("\n%s%d. %s", prefix, idx+1, name))
		}
		m.appendSys(b.String())
		return
	}

	target := strings.TrimSpace(strings.Join(args, " "))
	selectedModel := ""
	if idx, err := strconv.Atoi(target); err == nil {
		if idx >= 1 && idx <= len(models) {
			selectedModel = models[idx-1]
		}
	}
	if selectedModel == "" {
		for _, name := range models {
			if strings.EqualFold(name, target) {
				selectedModel = name
				break
			}
		}
	}
	if selectedModel == "" {
		m.appendErr(fmt.Sprintf("未找到 model: %s", target))
		return
	}

	req := buildSettingRequest(cur)
	req.AIService = config
	req.AIModelName = selectedModel
	req.SelectedModelName = selectedModel
	resp, err := m.client.UpdateSetting(req)
	if err != nil {
		m.appendErr(fmt.Sprintf("更新 model 失败: %v", err))
		return
	}
	m.invalidateDynamicCmdOptions()
	m.aiService = resp.AIService
	m.aiModel = resp.AIModelName
	m.appendSys(fmt.Sprintf("✓ Model 已更新: %s", selectedModel))
}

func (m *TUIModel) handleFocusModeCommand(args []string) {
	modes, err := m.client.ListAIFocusModes()
	if err != nil {
		m.appendErr(fmt.Sprintf("获取 focus mode 列表失败: %v", err))
		return
	}
	if len(modes) == 0 {
		m.appendSys("未获取到可用 focus mode")
		return
	}

	cur, err := m.client.GetSetting()
	if err != nil {
		m.appendErr(fmt.Sprintf("读取设置失败: %v", err))
		return
	}

	if len(args) == 0 {
		var b strings.Builder
		b.WriteString("可用 focus mode（用法: /focus_mode <name|序号>）:")
		for idx, item := range modes {
			prefix := "  "
			if strings.EqualFold(item.Name, cur.FocusModeLoop) {
				prefix = "* "
			}
			label := item.Name
			if item.VerboseNameZh != "" {
				label += "  (" + item.VerboseNameZh + ")"
			} else if item.VerboseName != "" {
				label += "  (" + item.VerboseName + ")"
			}
			b.WriteString(fmt.Sprintf("\n%s%d. %s", prefix, idx+1, label))
		}
		m.appendSys(b.String())
		return
	}

	target := strings.TrimSpace(strings.Join(args, " "))
	selected := ""
	if idx, err := strconv.Atoi(target); err == nil {
		if idx >= 1 && idx <= len(modes) {
			selected = modes[idx-1].Name
		}
	}
	if selected == "" {
		for _, item := range modes {
			if strings.EqualFold(item.Name, target) ||
				(item.VerboseName != "" && strings.EqualFold(item.VerboseName, target)) ||
				(item.VerboseNameZh != "" && strings.EqualFold(item.VerboseNameZh, target)) {
				selected = item.Name
				break
			}
		}
	}
	if selected == "" {
		m.appendErr(fmt.Sprintf("未找到 focus mode: %s", target))
		return
	}

	req := buildSettingRequest(cur)
	req.FocusModeLoop = selected
	if _, err := m.client.UpdateSetting(req); err != nil {
		m.appendErr(fmt.Sprintf("更新 focus mode 失败: %v", err))
		return
	}
	m.appendSys(fmt.Sprintf("✓ Focus Mode 已更新: %s", selected))
}

func (m *TUIModel) applySetting(key, val string) {
	cur, err := m.client.GetSetting()
	if err != nil {
		m.appendErr(err.Error())
		return
	}
	req := buildSettingRequest(cur)
	switch strings.ToLower(key) {
	case "service", "ai_service":
		req.AIService = val
	case "model", "ai_model":
		req.AIModelName = val
		req.SelectedModelName = val
	case "forge", "forge_name":
		req.ForgeName = val
	case "policy", "review_policy":
		req.ReviewPolicy = val
	case "provider", "selected_provider_id":
		req.SelectedProviderID = toInt64(val)
	case "selected_model", "selected_model_name":
		req.SelectedModelName = val
		req.AIModelName = val
	case "focus_mode", "focus_mode_loop":
		req.FocusModeLoop = val
	default:
		m.appendSys(fmt.Sprintf("未知设置键: %s (可选: service / model / forge / policy / provider / selected_model / focus_mode)", key))
		return
	}
	res, err := m.client.UpdateSetting(req)
	if err != nil {
		m.appendErr(err.Error())
		return
	}
	m.invalidateDynamicCmdOptions()
	m.aiService = res.AIService
	m.aiModel = res.AIModelName
	m.appendSys("✓ 设置已更新")
}

// ---------- layout ----------

func (m *TUIModel) relayout() {
	inputWidth := m.w - 4
	if inputWidth < 1 {
		inputWidth = 1
	}
	m.input.Width = inputWidth

	inputHeight := lipgloss.Height(sInputBar.Width(inputWidth).Render(m.input.View()))
	vpH := m.h - 2 - len(m.cmdMatches) - inputHeight
	if vpH < 1 {
		vpH = 1
	}
	if !m.ready {
		m.viewport = viewport.New(m.w, vpH)
		m.ready = true
	} else {
		m.viewport.Width = m.w
		m.viewport.Height = vpH
	}
	m.refreshVP()
}

func (m *TUIModel) refreshVP() {
	wasAtBottom := m.viewport.AtBottom()

	all := m.content
	if m.streamBuf != "" {
		switch {
		case m.streamNodeID == "re-act-loop-thought" || m.streamNodeID == "thought":
			all += sDim.Render(m.streamBuf)
		case isToolStdout(m.streamNodeID) || m.streamContentType == "log/tool":
			all += sDim.Render(m.streamBuf)
		default:
			all += m.streamBuf
		}
	}
	if m.phase == phaseConnecting || m.phase == phaseCreating || m.phase == phaseStreaming {
		if label := m.phaseLabel(); label != "" {
			all += "\n" + m.spin.View() + " " + sSystem.Render(label)
		}
	}
	m.viewport.SetContent(all)
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
}

// ---------- header / labels ----------

func (m TUIModel) headerLine() string {
	var parts []string
	parts = append(parts, "Yak AI Agent")
	switch m.phase {
	case phaseConnecting:
		parts = append(parts, "○ 连接中")
	case phaseReady:
		parts = append(parts, "● 就绪")
	case phaseCreating:
		parts = append(parts, "● 创建会话")
	case phaseStreaming:
		parts = append(parts, "● 运行中")
	case phaseInteractive:
		parts = append(parts, "● 等待输入")
	}
	if m.aiService != "" {
		parts = append(parts, fmt.Sprintf("[%s/%s]", m.aiService, m.aiModel))
	}
	if m.currentRunID != "" {
		parts = append(parts, shortID(m.currentRunID))
	}
	return strings.Join(parts, "  ")
}

func (m TUIModel) phaseLabel() string {
	return m.statusText
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12] + "…"
	}
	return id
}

// ---------- entry ----------

func RunTUI(client *AgentClient, serverURL string) error {
	m := NewTUIModel(client, serverURL)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
