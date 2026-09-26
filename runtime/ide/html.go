package ide

var ideHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Flang IDE</title>
<script src="https://cdn.tailwindcss.com"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/fabric.js/6.6.1/fabric.min.js"></script>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<script>tailwind.config={darkMode:'class',theme:{extend:{colors:{
  base:'#1e1e2e',mantle:'#181825',crust:'#11111b',
  surface0:'#313244',surface1:'#45475a',surface2:'#585b70',
  overlay0:'#6c7086',overlay1:'#7f849c',
  txt:'#cdd6f4',subtext:'#a6adc8',
  blue:'#89b4fa',green:'#a6e3a1',red:'#f38ba8',yellow:'#f9e2af',
  mauve:'#cba6f7',pink:'#f5c2e7',teal:'#94e2d5',peach:'#fab387',
  lavender:'#b4befe'
}}}}</script>
<style>
:root {
  --base: #1e1e2e;
  --mantle: #181825;
  --crust: #11111b;
  --surface0: #313244;
  --surface1: #45475a;
  --surface2: #585b70;
  --overlay0: #6c7086;
  --overlay1: #7f849c;
  --text: #cdd6f4;
  --subtext: #a6adc8;
  --blue: #89b4fa;
  --green: #a6e3a1;
  --red: #f38ba8;
  --yellow: #f9e2af;
  --mauve: #cba6f7;
  --pink: #f5c2e7;
  --teal: #94e2d5;
  --peach: #fab387;
  --lavender: #b4befe;
}

* { transition: background-color 0.15s, border-color 0.15s, color 0.15s; box-sizing: border-box; }
html, body { margin: 0; height: 100%; overflow: hidden; font-family: 'Inter', system-ui, -apple-system, sans-serif; }
body { background: var(--base); color: var(--text); }

::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--surface1); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: var(--surface2); }

/* Top toolbar */
.toolbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 16px; height: 48px; background: var(--mantle);
  border-bottom: 1px solid var(--surface0); flex-shrink: 0;
  backdrop-filter: blur(12px);
}
.toolbar-logo {
  display: flex; align-items: center; gap: 10px;
}
.toolbar-logo .logo-icon {
  width: 28px; height: 28px; border-radius: 8px;
  background: linear-gradient(135deg, #89b4fa22, #cba6f722);
  display: flex; align-items: center; justify-content: center;
  border: 1px solid var(--surface0);
}
.toolbar-logo .logo-text { font-weight: 700; font-size: 14px; color: var(--text); letter-spacing: -0.3px; }
.toolbar-logo .logo-version { font-size: 10px; color: var(--overlay0); font-weight: 500; }

/* Segmented control tabs */
.segmented-control {
  display: flex; align-items: center; gap: 2px;
  background: var(--crust); border-radius: 10px; padding: 3px;
  border: 1px solid var(--surface0);
}
.mode-tab {
  padding: 6px 18px; font-size: 12px; font-weight: 500;
  border-radius: 8px; cursor: pointer; background: none;
  border: none; color: var(--overlay0); transition: all 0.2s ease;
  font-family: 'Inter', system-ui, sans-serif;
}
.mode-tab:hover:not(.active) { color: var(--subtext); background: var(--surface0); }
.mode-tab.active {
  background: var(--blue); color: var(--crust);
  font-weight: 600; box-shadow: 0 2px 8px rgba(137,180,250,0.25);
}

/* Toolbar action buttons */
.toolbar-actions { display: flex; align-items: center; gap: 6px; }
.tool-btn {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 12px; font-size: 11px; font-weight: 500;
  border-radius: 8px; cursor: pointer; border: 1px solid var(--surface0);
  background: var(--crust); color: var(--subtext); transition: all 0.2s;
  font-family: 'Inter', system-ui, sans-serif;
}
.tool-btn:hover { background: var(--surface0); color: var(--text); border-color: var(--surface1); }
.tool-btn svg { width: 14px; height: 14px; }
.tool-btn.btn-run {
  background: var(--blue); color: var(--crust); border-color: var(--blue);
  font-weight: 600;
}
.tool-btn.btn-run:hover { background: #7ba8ed; }
.tool-btn.btn-stop:hover { background: rgba(243,139,168,0.15); color: var(--red); border-color: var(--red); }
.tool-btn.btn-preview.active { background: var(--blue); color: var(--crust); border-color: var(--blue); }

.save-indicator {
  display: none; align-items: center; gap: 4px;
  font-size: 10px; color: var(--yellow); font-weight: 500;
}
.save-indicator.visible { display: flex; }

/* File tree */
.file-tree { font-size: 12px; }
.file-item {
  display: flex; align-items: center; gap: 6px;
  padding: 5px 10px; border-radius: 6px; font-size: 12px;
  color: var(--subtext); cursor: pointer; margin: 1px 4px;
  transition: all 0.12s;
}
.file-item:hover { background: var(--surface0); color: var(--text); }
.file-item.active { background: rgba(137,180,250,0.1); color: var(--blue); font-weight: 500; }
.file-item.dir { font-weight: 500; color: var(--text); }
.file-children { padding-left: 14px; }

/* Tabs */
.tabs-bar {
  display: flex; background: var(--mantle); border-bottom: 1px solid var(--surface0);
  overflow-x: auto; flex-shrink: 0; min-height: 36px;
}
.tab {
  padding: 6px 14px; font-size: 12px; display: flex; align-items: center; gap: 6px;
  border-bottom: 2px solid transparent; color: var(--overlay0);
  cursor: pointer; white-space: nowrap; transition: all 0.15s;
  font-family: 'Inter', system-ui, sans-serif;
}
.tab:hover { color: var(--text); background: rgba(137,180,250,0.05); }
.tab.active { color: var(--blue); border-bottom-color: var(--blue); background: rgba(137,180,250,0.05); }
.tab .close {
  opacity: 0; font-size: 10px; padding: 2px 4px;
  border-radius: 4px; transition: all 0.12s;
}
.tab:hover .close { opacity: 0.5; }
.tab .close:hover { opacity: 1; background: rgba(243,139,168,0.2); color: var(--red); }

/* Panel headers */
.panel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.8px;
  color: var(--overlay0); border-bottom: 1px solid var(--surface0);
}
.panel-header-btn {
  padding: 4px; border-radius: 6px; background: none; border: none;
  color: var(--overlay0); cursor: pointer; display: flex;
  align-items: center; transition: all 0.15s;
}
.panel-header-btn:hover { background: var(--surface0); color: var(--text); }
.panel-header-btn svg { width: 14px; height: 14px; }

/* Terminal */
#terminal {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px; line-height: 1.7;
}
#terminal .line { padding: 0 14px; }
#terminal .error { color: var(--red); }
#terminal .success { color: var(--green); }
#terminal .info { color: var(--blue); }

/* Status bar */
.status-bar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 16px; height: 24px; font-size: 11px;
  background: var(--crust); border-top: 1px solid var(--surface0);
  flex-shrink: 0;
}
.status-bar .status-left, .status-bar .status-right {
  display: flex; align-items: center; gap: 12px;
}
.status-bar .status-right { color: var(--overlay0); }

/* Designer palette: compact grid */
.palette-grid {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 4px; padding: 8px;
}
.palette-icon {
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  padding: 10px 4px 8px; border-radius: 8px; cursor: grab;
  font-size: 10px; font-weight: 500; color: var(--overlay0);
  transition: all 0.15s; border: 1px solid transparent;
  background: transparent;
}
.palette-icon:hover {
  background: var(--surface0); color: var(--text);
  border-color: var(--surface1);
}
.palette-icon:active { cursor: grabbing; transform: scale(0.95); }
.palette-icon svg, .palette-icon .p-ico { width: 20px; height: 20px; display: flex; align-items: center; justify-content: center; }
.palette-icon .p-ico { font-size: 16px; line-height: 1; }

/* Designer canvas component cards */
.canvas-comp {
  padding: 12px 16px; margin-bottom: 8px; border-radius: 10px;
  border: 1px solid var(--surface0); background: var(--crust);
  cursor: pointer; transition: all 0.2s; position: relative;
  border-left: 3px solid var(--blue);
}
.canvas-comp:hover { border-color: rgba(137,180,250,0.4); background: rgba(137,180,250,0.05); }
.canvas-comp.selected { border-color: var(--blue); box-shadow: 0 0 0 2px rgba(137,180,250,0.2); }
.canvas-comp .comp-delete {
  position: absolute; top: 6px; right: 6px; width: 20px; height: 20px;
  border-radius: 6px; background: rgba(243,139,168,0.1); color: var(--red);
  border: none; cursor: pointer; font-size: 10px;
  display: none; align-items: center; justify-content: center;
}
.canvas-comp:hover .comp-delete { display: flex; }
.canvas-comp .comp-label {
  font-size: 10px; text-transform: uppercase; letter-spacing: 0.5px;
  font-weight: 600; margin-bottom: 4px;
}
.canvas-comp .comp-preview { font-size: 12px; color: var(--subtext); }

/* Properties panel */
.prop-section {
  padding: 12px 14px; border-bottom: 1px solid var(--surface0);
}
.prop-section-title {
  font-size: 10px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.6px; color: var(--overlay0); margin-bottom: 10px;
}
.prop-field label {
  display: block; font-size: 11px; color: var(--overlay1);
  margin-bottom: 4px; font-weight: 500;
}
.prop-input {
  width: 100%; background: var(--crust); border: 1px solid var(--surface0);
  border-radius: 6px; padding: 6px 10px; font-size: 12px;
  color: var(--text); outline: none; transition: border-color 0.15s;
  font-family: 'Inter', system-ui, sans-serif;
}
.prop-input:focus { border-color: var(--blue); }
.prop-input::placeholder { color: var(--surface2); }

/* Position/size grid */
.pos-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 6px;
}
.pos-grid .pos-cell {
  display: flex; align-items: center; gap: 4px;
}
.pos-grid .pos-label {
  font-size: 10px; font-weight: 600; color: var(--overlay0); width: 14px;
}
.pos-grid .pos-val {
  font-size: 11px; color: var(--subtext); font-family: 'JetBrains Mono', monospace;
}

/* Action buttons in props */
.prop-action-btn {
  width: 100%; padding: 6px 10px; font-size: 11px; font-weight: 500;
  border-radius: 6px; border: 1px solid var(--surface0); cursor: pointer;
  background: var(--crust); color: var(--subtext); transition: all 0.15s;
  font-family: 'Inter', system-ui, sans-serif;
}
.prop-action-btn:hover { background: var(--surface0); color: var(--text); }
.prop-action-btn.danger { border-color: rgba(243,139,168,0.2); color: var(--red); }
.prop-action-btn.danger:hover { background: rgba(243,139,168,0.1); }
.prop-action-btn.primary { background: var(--blue); color: var(--crust); border-color: var(--blue); font-weight: 600; }
.prop-action-btn.primary:hover { background: #7ba8ed; }

/* Flow editor nodes */
.flow-palette-item {
  display: flex; align-items: center; gap: 8px;
  padding: 7px 10px; font-size: 11px; border-radius: 6px;
  cursor: grab; background: transparent; color: var(--subtext);
  transition: all 0.15s; border-left: 3px solid transparent;
}
.flow-palette-item:hover { background: var(--surface0); color: var(--text); }
.flow-palette-item:active { cursor: grabbing; }

.flow-node {
  position: absolute; min-width: 170px; background: var(--mantle);
  border: 1px solid var(--surface0); border-radius: 10px;
  cursor: move; user-select: none; z-index: 10;
  box-shadow: 0 4px 16px rgba(0,0,0,0.3);
}
.flow-node.selected { border-color: var(--blue); box-shadow: 0 0 0 2px rgba(137,180,250,0.3), 0 4px 16px rgba(0,0,0,0.3); }
.flow-node .node-head {
  padding: 8px 12px; font-size: 11px; font-weight: 600;
  border-bottom: 1px solid var(--surface0);
  display: flex; align-items: center; justify-content: space-between;
  border-radius: 10px 10px 0 0;
}
.flow-node .node-body { padding: 8px 12px; font-size: 11px; color: var(--subtext); }
.flow-port {
  width: 12px; height: 12px; border-radius: 50%;
  border: 2px solid var(--blue); background: var(--crust);
  cursor: crosshair; position: absolute; z-index: 20;
  transition: all 0.15s;
}
.flow-port:hover { background: var(--blue); transform: scale(1.2); }
.flow-port.port-in { top: 50%; left: -6px; transform: translateY(-50%); }
.flow-port.port-out { top: 50%; right: -6px; transform: translateY(-50%); }
.flow-port:hover.port-in { transform: translateY(-50%) scale(1.2); }
.flow-port:hover.port-out { transform: translateY(-50%) scale(1.2); }

/* Layers list */
.layer-item {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 10px; border-radius: 6px; font-size: 11px;
  color: var(--subtext); cursor: pointer; transition: all 0.12s;
}
.layer-item:hover { background: var(--surface0); color: var(--text); }
.layer-item.active { background: rgba(137,180,250,0.1); color: var(--blue); }
.layer-item .layer-dot {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
}

/* Canvas toolbar overlays */
.canvas-toolbar {
  position: absolute; top: 12px; left: 12px; z-index: 10;
  display: flex; align-items: center; gap: 4px;
  background: var(--mantle); border: 1px solid var(--surface0);
  border-radius: 8px; padding: 4px;
}
.canvas-toolbar button {
  padding: 4px 8px; font-size: 11px; color: var(--overlay0);
  background: none; border: none; border-radius: 6px;
  cursor: pointer; transition: all 0.15s; font-family: 'Inter', system-ui, sans-serif;
}
.canvas-toolbar button:hover { background: var(--surface0); color: var(--text); }
.canvas-toolbar .sep {
  width: 1px; height: 16px; background: var(--surface0); margin: 0 2px;
}
.canvas-toolbar .zoom-text {
  font-size: 11px; color: var(--overlay0); padding: 0 6px;
  font-family: 'JetBrains Mono', monospace; min-width: 38px; text-align: center;
}

/* Dot grid on canvas */
.dot-grid {
  position: absolute; inset: 0; opacity: 0.15; pointer-events: none;
  background-image: radial-gradient(circle, var(--surface1) 1px, transparent 1px);
  background-size: 24px 24px;
}

/* Generated code block */
.code-output {
  font-family: 'JetBrains Mono', monospace; font-size: 11px;
  background: var(--crust); color: var(--green); padding: 12px;
  border-radius: 8px; overflow: auto; white-space: pre-wrap;
  border: 1px solid var(--surface0); max-height: 200px;
}

/* Terminal panel collapsible */
.terminal-panel {
  border-top: 1px solid var(--surface0); background: var(--mantle);
  flex-shrink: 0; display: flex; flex-direction: column;
}
.terminal-panel.collapsed { height: 32px !important; }
.terminal-panel.collapsed #terminal { display: none; }

/* Preview panel */
.preview-panel {
  border-top: 1px solid var(--surface0); background: var(--mantle);
  flex-shrink: 0;
}

/* Sidebar panel */
.sidebar-panel {
  background: var(--mantle); border-right: 1px solid var(--surface0);
  display: flex; flex-direction: column; flex-shrink: 0; overflow: hidden;
}
.sidebar-right {
  background: var(--mantle); border-left: 1px solid var(--surface0);
  display: flex; flex-direction: column; flex-shrink: 0; overflow: hidden;
}

/* Smooth resize handle */
.resize-handle {
  width: 3px; cursor: col-resize; background: transparent;
  transition: background 0.2s; flex-shrink: 0;
}
.resize-handle:hover { background: var(--blue); }
</style>
</head>
<body class="flex flex-col h-screen">

<!-- Top Toolbar -->
<div class="toolbar">
  <div class="flex items-center gap-4">
    <div class="toolbar-logo">
      <div class="logo-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="#89b4fa" stroke-width="2.5" style="width:16px;height:16px"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
      </div>
      <span class="logo-text">Flang IDE</span>
      <span class="logo-version">v0.5.1</span>
    </div>
    <div class="segmented-control">
      <button onclick="switchMode('editor')" id="mode-editor" class="mode-tab active">Codigo</button>
      <button onclick="switchMode('designer')" id="mode-designer" class="mode-tab">Designer</button>
      <button onclick="switchMode('fluxos')" id="mode-fluxos" class="mode-tab">Fluxos</button>
    </div>
  </div>
  <div class="toolbar-actions">
    <div id="save-indicator" class="save-indicator">
      <svg viewBox="0 0 24 24" fill="currentColor" style="width:8px;height:8px"><circle cx="12" cy="12" r="6"/></svg>
      Nao salvo
    </div>
    <button onclick="checkProject()" class="tool-btn" title="Verificar sintaxe">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
      Check
    </button>
    <button onclick="runProject()" class="tool-btn btn-run" title="Executar app">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
      Run
    </button>
    <button onclick="stopProject()" class="tool-btn btn-stop" title="Parar app">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="6" width="12" height="12" rx="1"/></svg>
      Stop
    </button>
    <button onclick="togglePreview()" id="btn-preview" class="tool-btn btn-preview" title="Preview ao vivo">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
      Preview
    </button>
  </div>
</div>

<!-- Main layout -->
<div class="flex flex-1 overflow-hidden">

  <!-- Left sidebar: File tree (shown in editor mode) -->
  <div id="sidebar-files" class="sidebar-panel" style="width:220px">
    <div class="panel-header">
      <span>Arquivos</span>
      <div class="flex items-center gap-1">
        <button onclick="createFile()" class="panel-header-btn" title="Novo arquivo">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        </button>
        <button onclick="loadFileTree()" class="panel-header-btn" title="Atualizar">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
        </button>
      </div>
    </div>
    <div id="file-tree" class="file-tree flex-1 overflow-y-auto py-1"></div>
  </div>

  <!-- Left sidebar: Designer palette + layers (shown in designer mode) -->
  <div id="sidebar-designer" class="sidebar-panel" style="width:200px;display:none">
    <div class="panel-header">
      <span>Componentes</span>
    </div>
    <div class="flex-1 overflow-y-auto">
      <div class="palette-grid">
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'titulo')">
          <div class="p-ico" style="color:#cba6f7;font-weight:700">T</div>
          <span>Titulo</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'lista')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#89b4fa" stroke-width="1.5" style="width:20px;height:20px"><line x1="3" y1="5" x2="17" y2="5"/><line x1="3" y1="10" x2="17" y2="10"/><line x1="3" y1="15" x2="17" y2="15"/></svg>
          <span>Lista</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'botao')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#a6e3a1" stroke-width="1.5" style="width:20px;height:20px"><rect x="2" y="5" width="16" height="10" rx="3"/></svg>
          <span>Botao</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'busca')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#94e2d5" stroke-width="1.5" style="width:20px;height:20px"><circle cx="9" cy="9" r="5"/><line x1="13" y1="13" x2="17" y2="17"/></svg>
          <span>Busca</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'grafico')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#f9e2af" stroke-width="1.5" style="width:20px;height:20px"><rect x="3" y="10" width="3" height="7"/><rect x="8.5" y="6" width="3" height="11"/><rect x="14" y="3" width="3" height="14"/></svg>
          <span>Grafico</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'cards')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#cba6f7" stroke-width="1.5" style="width:20px;height:20px"><rect x="2" y="2" width="7" height="7" rx="1.5"/><rect x="11" y="2" width="7" height="7" rx="1.5"/><rect x="2" y="11" width="7" height="7" rx="1.5"/><rect x="11" y="11" width="7" height="7" rx="1.5"/></svg>
          <span>Cards</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'formulario')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#f5c2e7" stroke-width="1.5" style="width:20px;height:20px"><rect x="3" y="2" width="14" height="16" rx="2"/><line x1="6" y1="6" x2="14" y2="6"/><line x1="6" y1="10" x2="14" y2="10"/><line x1="6" y1="14" x2="10" y2="14"/></svg>
          <span>Form</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'texto')">
          <div class="p-ico" style="color:#a6adc8;font-size:14px;font-weight:500">Aa</div>
          <span>Texto</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'imagem')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#94e2d5" stroke-width="1.5" style="width:20px;height:20px"><rect x="2" y="2" width="16" height="16" rx="2"/><circle cx="7" cy="7" r="1.5"/><polyline points="18 13 13 8 4 18"/></svg>
          <span>Imagem</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'separador')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#585b70" stroke-width="2" style="width:20px;height:20px"><line x1="2" y1="10" x2="18" y2="10"/></svg>
          <span>Separador</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'input')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#b4befe" stroke-width="1.5" style="width:20px;height:20px"><rect x="2" y="5" width="16" height="10" rx="2"/><line x1="6" y1="8" x2="6" y2="12"/></svg>
          <span>Input</span>
        </div>
        <div class="palette-icon" draggable="true" ondragstart="dragStartComp(event,'select')">
          <svg viewBox="0 0 20 20" fill="none" stroke="#fab387" stroke-width="1.5" style="width:20px;height:20px"><rect x="2" y="5" width="16" height="10" rx="2"/><polyline points="12 9 14 11 16 9"/></svg>
          <span>Dropdown</span>
        </div>
      </div>

      <!-- Layers section -->
      <div style="border-top:1px solid var(--surface0);margin-top:4px">
        <div class="panel-header" style="border:none">
          <span>Camadas</span>
        </div>
        <div id="designer-layers" class="px-2 pb-2">
          <div style="padding:8px 10px;font-size:11px;color:var(--overlay0)">Nenhum componente</div>
        </div>
      </div>
    </div>

    <div style="border-top:1px solid var(--surface0);padding:10px 12px">
      <label style="font-size:10px;font-weight:600;color:var(--overlay0);text-transform:uppercase;letter-spacing:0.5px;display:block;margin-bottom:6px">Tela</label>
      <input type="text" id="d-screen-name" value="principal" placeholder="Nome da tela"
        class="prop-input" style="margin-bottom:8px" oninput="updateDesignerCode()">
      <div class="flex gap-2">
        <button onclick="clearCanvas()" class="prop-action-btn" style="flex:1">Limpar</button>
        <button onclick="generateFromDesigner()" class="prop-action-btn primary" style="flex:1">Salvar .ge</button>
      </div>
    </div>
  </div>

  <!-- Left sidebar: Fluxos palette (shown in fluxos mode) -->
  <div id="sidebar-fluxos" class="sidebar-panel" style="width:200px;display:none">
    <div class="panel-header">
      <span>Nodos</span>
    </div>
    <div class="flex-1 overflow-y-auto p-2">
      <div style="font-size:10px;font-weight:600;color:var(--overlay0);text-transform:uppercase;letter-spacing:0.6px;padding:6px 8px;margin-bottom:2px">Gatilhos</div>
      <div class="space-y-1 mb-4">
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'trigger_click')" style="border-left-color:var(--green)">Quando Clicar</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'trigger_create')" style="border-left-color:var(--green)">Quando Criar</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'trigger_update')" style="border-left-color:var(--green)">Quando Atualizar</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'trigger_cron')" style="border-left-color:var(--green)">Agendamento</div>
      </div>

      <div style="font-size:10px;font-weight:600;color:var(--overlay0);text-transform:uppercase;letter-spacing:0.6px;padding:6px 8px;margin-bottom:2px">Acoes</div>
      <div class="space-y-1 mb-4">
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'action_create')" style="border-left-color:var(--blue)">Criar Registro</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'action_update')" style="border-left-color:var(--blue)">Atualizar Registro</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'action_delete')" style="border-left-color:var(--blue)">Deletar Registro</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'action_message')" style="border-left-color:var(--blue)">Mostrar Mensagem</div>
      </div>

      <div style="font-size:10px;font-weight:600;color:var(--overlay0);text-transform:uppercase;letter-spacing:0.6px;padding:6px 8px;margin-bottom:2px">Integracoes</div>
      <div class="space-y-1 mb-4">
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'integ_whatsapp')" style="border-left-color:var(--green)">Enviar WhatsApp</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'integ_email')" style="border-left-color:var(--peach)">Enviar Email</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'integ_http')" style="border-left-color:var(--teal)">Chamar API</div>
      </div>

      <div style="font-size:10px;font-weight:600;color:var(--overlay0);text-transform:uppercase;letter-spacing:0.6px;padding:6px 8px;margin-bottom:2px">Logica</div>
      <div class="space-y-1 mb-4">
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'logic_if')" style="border-left-color:var(--yellow)">Condicao (Se)</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'logic_loop')" style="border-left-color:var(--yellow)">Para Cada</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'logic_set')" style="border-left-color:var(--yellow)">Definir Variavel</div>
        <div class="flow-palette-item" draggable="true" ondragstart="dragNode(event,'logic_function')" style="border-left-color:var(--yellow)">Funcao</div>
      </div>
    </div>

    <div style="border-top:1px solid var(--surface0);padding:10px 12px">
      <button onclick="generateFromFlow()" class="prop-action-btn primary" style="width:100%;margin-bottom:6px">Gerar Codigo .ge</button>
      <button onclick="clearFlow()" class="prop-action-btn" style="width:100%">Limpar Fluxo</button>
    </div>
  </div>

  <!-- Panel: Editor (code) -->
  <div id="panel-editor" class="flex-1 flex flex-col overflow-hidden">
    <!-- Tabs -->
    <div id="tabs" class="tabs-bar"></div>

    <!-- Monaco Editor container -->
    <div id="editor-container" class="flex-1 overflow-hidden" style="background:var(--base)"></div>

    <!-- Terminal panel -->
    <div id="terminal-panel" class="terminal-panel" style="height:180px">
      <div class="panel-header" style="cursor:pointer" onclick="toggleTerminal()">
        <div class="flex items-center gap-2">
          <span>Terminal</span>
          <svg id="terminal-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:12px;height:12px;transition:transform 0.2s"><polyline points="6 9 12 15 18 9"/></svg>
        </div>
        <button onclick="event.stopPropagation();clearTerminal()" class="panel-header-btn" title="Limpar">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
      <div id="terminal" class="overflow-y-auto flex-1 py-1" style="height:calc(100% - 36px)"></div>
    </div>
  </div>

  <!-- Panel: Designer (Fabric.js canvas editor) -->
  <div id="panel-designer" style="display:none" class="flex flex-1 overflow-hidden">
    <!-- Fabric.js Canvas -->
    <div class="flex-1 relative overflow-hidden" id="canvas-wrapper" style="background:var(--crust)"
         ondragover="event.preventDefault();this.style.boxShadow='inset 0 0 0 2px var(--blue)'"
         ondragleave="this.style.boxShadow='none'"
         ondrop="dropOnCanvas(event);this.style.boxShadow='none'">
      <div class="dot-grid"></div>
      <div class="canvas-toolbar">
        <span id="zoom-level" class="zoom-text">100%</span>
        <div class="sep"></div>
        <button onclick="zoomIn()" title="Zoom in">+</button>
        <button onclick="zoomOut()" title="Zoom out">-</button>
        <button onclick="zoomReset()" title="Reset zoom">Fit</button>
        <div class="sep"></div>
        <button onclick="undoCanvas()" title="Desfazer (Ctrl+Z)">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
        </button>
        <button onclick="redoCanvas()" title="Refazer (Ctrl+Y)">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
        </button>
      </div>
      <canvas id="fabric-canvas"></canvas>
    </div>

    <!-- Properties + Code -->
    <div class="sidebar-right" style="width:260px">
      <div class="flex-1 overflow-y-auto">
        <div class="prop-section">
          <div class="prop-section-title">Propriedades</div>
          <div id="d-props">
            <p style="font-size:11px;color:var(--overlay0)">Clique num componente no canvas</p>
          </div>
        </div>
      </div>
      <div style="border-top:1px solid var(--surface0);max-height:40%;overflow-y:auto">
        <div class="prop-section" style="border:none">
          <div class="prop-section-title">Codigo .ge Gerado</div>
          <pre id="d-generated" class="code-output"></pre>
        </div>
      </div>
    </div>
  </div>

  <!-- Panel: Fluxos (flow logic editor) -->
  <div id="panel-fluxos" style="display:none" class="flex flex-1 overflow-hidden">
    <!-- Flow Canvas -->
    <div class="flex-1 relative overflow-hidden" id="flow-canvas" style="background:var(--crust)"
         ondragover="event.preventDefault()"
         ondrop="dropNode(event)"
         onmousedown="flowCanvasMouseDown(event)"
         onmousemove="flowCanvasMouseMove(event)"
         onmouseup="flowCanvasMouseUp(event)">
      <svg id="flow-svg" class="absolute inset-0 w-full h-full pointer-events-none" style="z-index:1"></svg>
      <div id="flow-nodes" class="absolute inset-0" style="z-index:2"></div>
      <div class="dot-grid"></div>
    </div>

    <!-- Flow Properties -->
    <div class="sidebar-right" style="width:260px">
      <div class="flex-1 overflow-y-auto">
        <div class="prop-section">
          <div class="prop-section-title">Propriedades do Nodo</div>
          <div id="flow-props">
            <p style="font-size:11px;color:var(--overlay0)">Selecione um nodo</p>
          </div>
        </div>
      </div>
      <div style="border-top:1px solid var(--surface0);max-height:45%;overflow-y:auto">
        <div class="prop-section" style="border:none">
          <div class="prop-section-title">Codigo Gerado</div>
          <pre id="flow-generated-code" class="code-output"></pre>
        </div>
      </div>
    </div>
  </div>

</div>

<!-- Preview panel -->
<div id="preview-panel" class="preview-panel" style="display:none;height:45%">
  <div class="panel-header">
    <div class="flex items-center gap-3">
      <span>Preview</span>
      <input type="text" id="preview-url" value="http://localhost:8080" class="prop-input" style="width:200px;font-size:11px;padding:3px 8px" readonly>
    </div>
    <div class="flex items-center gap-1">
      <button onclick="refreshPreview()" class="panel-header-btn" title="Atualizar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
      </button>
      <button onclick="togglePreview()" class="panel-header-btn" title="Fechar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
  </div>
  <iframe id="preview-iframe" src="" style="width:100%;height:calc(100% - 36px);border:none;background:white"></iframe>
</div>

<!-- Status bar -->
<div class="status-bar">
  <div class="status-left">
    <div class="flex items-center gap-1.5">
      <svg viewBox="0 0 24 24" fill="none" stroke="#89b4fa" stroke-width="2" style="width:10px;height:10px"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
      <span style="font-weight:500;color:var(--blue)">Flang</span>
    </div>
    <span id="status-file" style="color:var(--overlay0)">Nenhum arquivo</span>
    <span id="status-modified" style="display:none;color:var(--yellow);font-size:10px">&#9679; Nao salvo</span>
  </div>
  <div class="status-right">
    <span id="status-lang">Flang (.ge)</span>
    <span id="status-cursor">Ln 1, Col 1</span>
  </div>
</div>

<!-- Monaco Editor from CDN -->
<script src="https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/loader.min.js"></script>
<script>
// State
var editor = null;
var openFiles = {};
var activeFile = null;
var modified = {};

// Toggle terminal collapse
function toggleTerminal() {
  var panel = document.getElementById('terminal-panel');
  var chevron = document.getElementById('terminal-chevron');
  panel.classList.toggle('collapsed');
  if (panel.classList.contains('collapsed')) {
    chevron.style.transform = 'rotate(-90deg)';
  } else {
    chevron.style.transform = 'rotate(0deg)';
  }
}

// Mode switching
function switchMode(mode) {
  document.querySelectorAll('.mode-tab').forEach(function(t){t.classList.remove('active');});
  document.getElementById('mode-'+mode).classList.add('active');

  // Panels
  document.getElementById('panel-editor').style.display = mode==='editor' ? 'flex' : 'none';
  document.getElementById('panel-designer').style.display = mode==='designer' ? 'flex' : 'none';
  document.getElementById('panel-fluxos').style.display = mode==='fluxos' ? 'flex' : 'none';

  // Sidebars
  document.getElementById('sidebar-files').style.display = mode==='editor' ? 'flex' : 'none';
  document.getElementById('sidebar-designer').style.display = mode==='designer' ? 'flex' : 'none';
  document.getElementById('sidebar-fluxos').style.display = mode==='fluxos' ? 'flex' : 'none';

  if (mode === 'designer') {
    setTimeout(function() { initFabricCanvas(); }, 100);
  }
}

// Initialize Monaco
require.config({ paths: { vs: 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs' }});
require(['vs/editor/editor.main'], function() {

  // Register Flang language
  monaco.languages.register({ id: 'flang', extensions: ['.ge'] });

  // Syntax highlighting
  monaco.languages.setMonarchTokensProvider('flang', {
    keywords: ['sistema','dados','telas','tela','eventos','tema','logica','banco','autenticacao','integracoes',
      'importar','de','system','models','screens','screen','events','theme','logic','database','auth','import','from',
      'rotas','rota','paginas','pagina','sidebar','item'],
    typeKeywords: ['texto','texto_longo','numero','dinheiro','email','telefone','data','booleano','imagem','arquivo',
      'upload','link','status','senha','enum','text','number','money','phone','image','file','password','boolean','date'],
    controlKeywords: ['se','senao','enquanto','repetir','para','para_cada','funcao','retornar','definir','mostrar',
      'quando','clicar','criar','atualizar','deletar','enviar','validar','tentar','erro','parar','continuar',
      'if','else','while','repeat','for','function','return','set','print','when','click','create','update','delete','try','error','break','continue'],
    modifiers: ['obrigatorio','unico','pertence_a','tem_muitos','muitos_para_muitos','soft_delete','indice','padrao',
      'required','unique','belongs_to','has_many','many_to_many','index','default'],
    screenKw: ['titulo','lista','mostrar','botao','busca','grafico','dashboard','formulario','tabela','requer','publico',
      'title','list','show','button','search','chart','form','table','requires','public'],
    themeKw: ['cor','primaria','secundaria','destaque','escuro','claro','fonte','borda','fundo','estilo','icone',
      'moderno','simples','elegante','corporativo','glassmorphism','flat','neumorphism','minimal'],
    colors: ['azul','verde','vermelho','roxo','laranja','rosa','amarelo','ciano','indigo','branco','preto'],
    operators: ['==','!=','>=','<=','>','<','+','-','*','/','='],
    tokenizer: {
      root: [
        [/#.*$/, 'comment'],
        [/\/\/.*$/, 'comment'],
        [/"[^"]*"/, 'string'],
        [/\b\d+(\.\d+)?\b/, 'number'],
        [/\b(verdadeiro|falso|nulo|nada|true|false|null)\b/, 'constant'],
        [/[a-zA-Z_\u00C0-\u024F\u0400-\u04FF\u4E00-\u9FFF\u3040-\u309F\u30A0-\u30FF\uAC00-\uD7AF\u0600-\u06FF\u0900-\u097F\u0980-\u09FF\u0E00-\u0E7F][a-zA-Z0-9_\u00C0-\u024F\u0400-\u04FF\u4E00-\u9FFF\u3040-\u309F\u30A0-\u30FF\uAC00-\uD7AF\u0600-\u06FF\u0900-\u097F\u0980-\u09FF\u0E00-\u0E7F]*/, {
          cases: {
            '@keywords': 'keyword',
            '@typeKeywords': 'type',
            '@controlKeywords': 'keyword.control',
            '@modifiers': 'keyword.modifier',
            '@screenKw': 'keyword.screen',
            '@themeKw': 'keyword.theme',
            '@colors': 'constant.color',
            '@default': 'identifier'
          }
        }],
        [/[{}()\[\]]/, 'delimiter'],
        [/[;,.]/, 'delimiter'],
        [/:/, 'delimiter.colon'],
      ]
    }
  });

  // Catppuccin Mocha theme
  monaco.editor.defineTheme('flang-dark', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: 'cba6f7', fontStyle: 'bold' },
      { token: 'keyword.control', foreground: 'f38ba8' },
      { token: 'keyword.modifier', foreground: 'fab387' },
      { token: 'keyword.screen', foreground: 'a6e3a1' },
      { token: 'keyword.theme', foreground: 'f5c2e7' },
      { token: 'type', foreground: '89b4fa', fontStyle: 'italic' },
      { token: 'string', foreground: 'a6e3a1' },
      { token: 'number', foreground: 'fab387' },
      { token: 'comment', foreground: '6c7086', fontStyle: 'italic' },
      { token: 'constant', foreground: 'f9e2af' },
      { token: 'constant.color', foreground: '94e2d5' },
      { token: 'delimiter.colon', foreground: '6c7086' },
      { token: 'identifier', foreground: 'cdd6f4' },
    ],
    colors: {
      'editor.background': '#1e1e2e',
      'editor.foreground': '#cdd6f4',
      'editor.lineHighlightBackground': '#31324410',
      'editor.selectionBackground': '#89b4fa30',
      'editorCursor.foreground': '#89b4fa',
      'editorLineNumber.foreground': '#45475a',
      'editorLineNumber.activeForeground': '#89b4fa',
      'editor.selectionHighlightBackground': '#89b4fa15',
      'editorWidget.background': '#181825',
      'editorSuggestWidget.background': '#181825',
      'editorSuggestWidget.border': '#313244',
      'editorSuggestWidget.selectedBackground': '#313244',
      'editorHoverWidget.background': '#181825',
      'editorHoverWidget.border': '#313244',
      'input.background': '#11111b',
      'input.border': '#313244',
      'focusBorder': '#89b4fa',
      'scrollbar.shadow': '#11111b',
      'scrollbarSlider.background': '#45475a40',
      'scrollbarSlider.hoverBackground': '#45475a80',
      'scrollbarSlider.activeBackground': '#585b70',
    }
  });

  // Create editor
  editor = monaco.editor.create(document.getElementById('editor-container'), {
    value: '# Bem-vindo ao Flang IDE!\\n#\\n# Para comecar:\\n#   1. Abra um arquivo .ge na arvore a esquerda\\n#   2. Ou clique em Designer para montar telas visualmente\\n#   3. Ou clique em Fluxos para criar logica visual\\n#\\n# Atalhos:\\n#   Ctrl+S     Salvar\\n#   Ctrl+Z     Desfazer (no Designer)\\n#   Ctrl+Y     Refazer (no Designer)\\n#   Delete     Remover componente selecionado\\n#\\n# Templates prontos (no terminal):\\n#   flang new loja\\n#   flang new clinica\\n#   flang new escola\\n#   flang new delivery\\n#   flang new crm\\n#   flang new helpdesk\\n#   flang new blog\\n#   flang new financeiro\\n#\\n# 60+ funcoes built-in, 25 tipos de dados, 20 idiomas\\n# Documentacao: docs/TUTORIAL.md\\n',
    language: 'flang',
    theme: 'flang-dark',
    fontSize: 14,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
    minimap: { enabled: true, scale: 1 },
    smoothScrolling: true,
    cursorBlinking: 'smooth',
    cursorSmoothCaretAnimation: 'on',
    padding: { top: 16, bottom: 16 },
    renderLineHighlight: 'all',
    bracketPairColorization: { enabled: true },
    automaticLayout: true,
    wordWrap: 'on',
    tabSize: 2,
    scrollBeyondLastLine: false,
    fontLigatures: true,
    lineHeight: 22,
    letterSpacing: 0.3,
  });

  // Track cursor position
  editor.onDidChangeCursorPosition(function(e) {
    document.getElementById('status-cursor').textContent = 'Ln ' + e.position.lineNumber + ', Col ' + e.position.column;
  });

  // Track modifications
  editor.onDidChangeModelContent(function() {
    if (activeFile) {
      modified[activeFile] = true;
      updateTabModified(activeFile, true);
      document.getElementById('status-modified').style.display = 'inline';
      document.getElementById('save-indicator').classList.add('visible');
    }
  });

  // Keyboard shortcuts
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, function() {
    saveCurrentFile();
  });

  // Load file tree
  loadFileTree();

  termLog('info', 'Flang IDE iniciado. Pronto para editar!');
});

// File tree
function loadFileTree() {
  fetch('/api/files').then(function(r){return r.json();}).then(function(files) {
    document.getElementById('file-tree').innerHTML = renderTree(files);
  });
}

function renderTree(files) {
  if (!files || !files.length) return '<div style="padding:12px 14px;font-size:11px;color:var(--overlay0)">Nenhum arquivo</div>';
  var html = '';
  files.forEach(function(f) {
    if (f.isDir) {
      html += '<div class="file-item dir" onclick="this.nextElementSibling.classList.toggle(\'hidden\')">'+
        '<svg viewBox="0 0 24 24" fill="none" stroke="#fab387" stroke-width="2" style="width:14px;height:14px;flex-shrink:0"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>'+
        '<span>'+f.name+'</span></div>';
      html += '<div class="file-children">' + renderTree(f.children) + '</div>';
    } else {
      var icon = f.name.endsWith('.ge') ?
        '<svg viewBox="0 0 24 24" fill="none" stroke="#89b4fa" stroke-width="2" style="width:14px;height:14px;flex-shrink:0"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>' :
        '<svg viewBox="0 0 24 24" fill="none" stroke="#6c7086" stroke-width="2" style="width:14px;height:14px;flex-shrink:0"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>';
      html += '<div class="file-item" onclick="openFile(\''+f.path+'\')" data-path="'+f.path+'">'+icon+'<span>'+f.name+'</span></div>';
    }
  });
  return html;
}

// Open file
function openFile(path) {
  // Highlight in tree
  document.querySelectorAll('.file-item').forEach(function(el){el.classList.remove('active');});
  var el = document.querySelector('[data-path="'+path+'"]');
  if(el) el.classList.add('active');

  if (openFiles[path]) {
    switchToFile(path);
    return;
  }

  fetch('/api/file?path='+encodeURIComponent(path)).then(function(r){return r.text();}).then(function(content) {
    openFiles[path] = content;
    addTab(path);
    switchToFile(path);
    document.getElementById('status-file').textContent = path;
  });
}

function switchToFile(path) {
  activeFile = path;
  var lang = path.endsWith('.ge') ? 'flang' : (path.endsWith('.json') ? 'json' : (path.endsWith('.go') ? 'go' : (path.endsWith('.js') ? 'javascript' : 'plaintext')));
  editor.setValue(openFiles[path] || '');
  monaco.editor.setModelLanguage(editor.getModel(), lang);

  // Update tabs
  document.querySelectorAll('.tab').forEach(function(t){t.classList.remove('active');});
  var tab = document.querySelector('.tab[data-path="'+path+'"]');
  if(tab) tab.classList.add('active');

  // Update tree highlight
  document.querySelectorAll('.file-item').forEach(function(el){el.classList.remove('active');});
  var el = document.querySelector('[data-path="'+path+'"]');
  if(el) el.classList.add('active');

  document.getElementById('status-file').textContent = path;
  document.getElementById('status-lang').textContent = lang === 'flang' ? 'Flang (.ge)' : lang;
}

// Tabs
function addTab(path) {
  var name = path.split('/').pop();
  var tabs = document.getElementById('tabs');
  if (document.querySelector('.tab[data-path="'+path+'"]')) return;

  var tab = document.createElement('div');
  tab.className = 'tab active';
  tab.setAttribute('data-path', path);
  tab.innerHTML = '<span onclick="switchToFile(\''+path+'\')">'+name+'</span><span class="close" onclick="event.stopPropagation();closeTab(\''+path+'\')">&times;</span>';
  tab.onclick = function(){switchToFile(path);};
  tabs.appendChild(tab);

  document.querySelectorAll('.tab').forEach(function(t){t.classList.remove('active');});
  tab.classList.add('active');
}

function closeTab(path) {
  var tab = document.querySelector('.tab[data-path="'+path+'"]');
  if(tab) tab.remove();
  delete openFiles[path];
  delete modified[path];

  var remaining = document.querySelectorAll('.tab');
  if (remaining.length > 0) {
    var last = remaining[remaining.length-1];
    switchToFile(last.getAttribute('data-path'));
  } else {
    activeFile = null;
    editor.setValue('# Selecione um arquivo');
    document.getElementById('status-file').textContent = 'Nenhum arquivo';
  }
}

function updateTabModified(path, isModified) {
  var tab = document.querySelector('.tab[data-path="'+path+'"] span:first-child');
  if (!tab) return;
  var name = path.split('/').pop();
  tab.textContent = isModified ? name + ' \u25cf' : name;
}

// Save
function saveCurrentFile() {
  if (!activeFile) return;
  var content = editor.getValue();
  openFiles[activeFile] = content;

  fetch('/api/file/save', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({path: activeFile, content: content})
  }).then(function(r) {
    if (r.ok) {
      modified[activeFile] = false;
      updateTabModified(activeFile, false);
      document.getElementById('status-modified').style.display = 'none';
      document.getElementById('save-indicator').classList.remove('visible');
      termLog('success', 'Salvo: ' + activeFile);
    }
  });
}

// Create file
function createFile() {
  var name = prompt('Nome do arquivo (ex: dados/produto.ge):');
  if (!name) return;
  fetch('/api/file/create', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({path: name, isDir: false})
  }).then(function() {
    loadFileTree();
    setTimeout(function(){openFile(name);}, 500);
  });
}

// Run/Check/Stop
function runProject() {
  var file = findMainFile();
  termLog('info', 'Executando ' + file + '...');
  fetch('/api/run', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({file: file})
  }).then(function(r){return r.json();}).then(function(d) {
    if (d.status === 'running') {
      termLog('success', 'App rodando em ' + d.url);
      if (!previewOpen) togglePreview();
      setTimeout(function(){ refreshPreview(); }, 2000);
    } else {
      termLog('error', 'Erro: ' + (d.message||'desconhecido'));
    }
  });
}

function stopProject() {
  fetch('/api/stop').then(function(r){return r.json();}).then(function() {
    termLog('info', 'App parado.');
  });
}

function checkProject() {
  var file = findMainFile();
  termLog('info', 'Verificando ' + file + '...');
  fetch('/api/check', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({file: file})
  }).then(function(r){return r.json();}).then(function(d) {
    var lines = d.output.split('\n');
    lines.forEach(function(line) {
      if (!line.trim()) return;
      if (line.indexOf('ERRO') >= 0) termLog('error', line);
      else if (line.indexOf('valido') >= 0 || line.indexOf('OK') >= 0) termLog('success', line);
      else termLog('info', line);
    });
  });
}

function findMainFile() {
  if (activeFile && activeFile.endsWith('.ge')) return activeFile;
  return 'inicio.ge';
}

// Terminal
function termLog(type, msg) {
  var term = document.getElementById('terminal');
  var time = new Date().toLocaleTimeString();
  term.innerHTML += '<div class="line ' + type + '"><span style="opacity:0.35;margin-right:6px">['+time+']</span> '+msg+'</div>';
  term.scrollTop = term.scrollHeight;
}

function clearTerminal() {
  document.getElementById('terminal').innerHTML = '';
}

// ============================================================
// DESIGNER - Fabric.js Canvas Editor
// ============================================================

var fabricCanvas = null;
var designerReady = false;
var compCounter = 0;

function initFabricCanvas() {
  if (designerReady) return;

  var wrapper = document.getElementById('canvas-wrapper');
  var cEl = document.getElementById('fabric-canvas');
  cEl.width = wrapper.clientWidth;
  cEl.height = wrapper.clientHeight;

  fabricCanvas = new fabric.Canvas('fabric-canvas', {
    backgroundColor: '#11111b',
    selection: true,
    preserveObjectStacking: true,
  });

  // Zoom with mouse wheel
  fabricCanvas.on('mouse:wheel', function(opt) {
    var delta = opt.e.deltaY;
    var zoom = fabricCanvas.getZoom();
    zoom *= Math.pow(0.999, delta);
    if (zoom > 5) zoom = 5;
    if (zoom < 0.2) zoom = 0.2;
    fabricCanvas.zoomToPoint({ x: opt.e.offsetX, y: opt.e.offsetY }, zoom);
    document.getElementById('zoom-level').textContent = Math.round(zoom * 100) + '%';
    opt.e.preventDefault();
    opt.e.stopPropagation();
  });

  // Pan with Alt+drag or middle mouse
  var panning = false;
  fabricCanvas.on('mouse:down', function(opt) {
    if (opt.e.altKey || opt.e.button === 1) {
      panning = true;
      fabricCanvas.selection = false;
      fabricCanvas.setCursor('grab');
    }
  });
  fabricCanvas.on('mouse:move', function(opt) {
    if (panning) {
      var e = opt.e;
      var vpt = fabricCanvas.viewportTransform;
      vpt[4] += e.movementX;
      vpt[5] += e.movementY;
      fabricCanvas.requestRenderAll();
    }
  });
  fabricCanvas.on('mouse:up', function() {
    panning = false;
    fabricCanvas.selection = true;
    fabricCanvas.setCursor('default');
  });

  // Select/deselect
  fabricCanvas.on('selection:created', function(e) { showDesignerProps(e.selected[0]); updateLayersList(); });
  fabricCanvas.on('selection:updated', function(e) { showDesignerProps(e.selected[0]); updateLayersList(); });
  fabricCanvas.on('selection:cleared', function() { clearDesignerProps(); updateLayersList(); });

  // Update code on move/resize
  fabricCanvas.on('object:modified', function() { updateDesignerCode(); saveCanvasState(); });
  fabricCanvas.on('object:added', function() { updateDesignerCode(); saveCanvasState(); updateLayersList(); });
  fabricCanvas.on('object:removed', function() { updateDesignerCode(); saveCanvasState(); updateLayersList(); });

  // Keyboard shortcuts for designer
  document.addEventListener('keydown', function(e) {
    // Ctrl+Z = undo
    if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey && document.getElementById('panel-designer').style.display !== 'none') {
      e.preventDefault();
      undoCanvas();
      return;
    }
    // Ctrl+Shift+Z or Ctrl+Y = redo
    if ((e.ctrlKey || e.metaKey) && (e.key === 'y' || (e.key === 'z' && e.shiftKey)) && document.getElementById('panel-designer').style.display !== 'none') {
      e.preventDefault();
      redoCanvas();
      return;
    }
    // Delete key
    if (e.key === 'Delete' && fabricCanvas && fabricCanvas.getActiveObject()) {
      var obj = fabricCanvas.getActiveObject();
      if (obj && obj.compType) {
        fabricCanvas.remove(obj);
        fabricCanvas.discardActiveObject();
        clearDesignerProps();
        updateDesignerCode();
      }
    }
  });

  // Resize observer
  new ResizeObserver(function() {
    if (!fabricCanvas) return;
    fabricCanvas.setDimensions({
      width: wrapper.clientWidth,
      height: wrapper.clientHeight
    });
    fabricCanvas.renderAll();
  }).observe(wrapper);

  designerReady = true;
}

// Update layers list in sidebar
function updateLayersList() {
  var container = document.getElementById('designer-layers');
  if (!fabricCanvas) {
    container.innerHTML = '<div style="padding:8px 10px;font-size:11px;color:var(--overlay0)">Nenhum componente</div>';
    return;
  }
  var objects = fabricCanvas.getObjects().filter(function(o){ return o.compType; });
  if (objects.length === 0) {
    container.innerHTML = '<div style="padding:8px 10px;font-size:11px;color:var(--overlay0)">Nenhum componente</div>';
    return;
  }
  var activeObj = fabricCanvas.getActiveObject();
  var html = '';
  objects.forEach(function(obj, i) {
    var isActive = activeObj === obj;
    html += '<div class="layer-item'+(isActive?' active':'')+'" onclick="selectLayerItem('+i+')">';
    html += '<div class="layer-dot" style="background:'+(obj.compColor||'var(--blue)')+'"></div>';
    html += '<span>'+(obj.compLabel||obj.compType)+'</span>';
    html += '</div>';
  });
  container.innerHTML = html;
}

function selectLayerItem(index) {
  if (!fabricCanvas) return;
  var objects = fabricCanvas.getObjects().filter(function(o){ return o.compType; });
  if (objects[index]) {
    fabricCanvas.setActiveObject(objects[index]);
    fabricCanvas.renderAll();
    showDesignerProps(objects[index]);
    updateLayersList();
  }
}

function dragStartComp(e, type) {
  e.dataTransfer.setData('compType', type);
  e.dataTransfer.effectAllowed = 'copy';
}

function dropOnCanvas(e) {
  e.preventDefault();
  var type = e.dataTransfer.getData('compType');
  if (!type) return;
  if (!fabricCanvas) initFabricCanvas();
  var rect = document.getElementById('canvas-wrapper').getBoundingClientRect();
  var zoom = fabricCanvas.getZoom();
  var vpt = fabricCanvas.viewportTransform;
  var x = (e.clientX - rect.left - vpt[4]) / zoom;
  var y = (e.clientY - rect.top - vpt[5]) / zoom;
  addToCanvas(type, x, y);
}

function addToCanvas(type, dropX, dropY) {
  if (!fabricCanvas) initFabricCanvas();

  compCounter++;
  var id = 'comp-' + compCounter;
  var group;

  var cx = dropX !== undefined ? dropX : fabricCanvas.width / 2 / fabricCanvas.getZoom();
  var cy = dropY !== undefined ? dropY : fabricCanvas.height / 2 / fabricCanvas.getZoom();
  if (dropX === undefined) {
    cx += (Math.random() - 0.5) * 100;
    cy += (Math.random() - 0.5) * 100;
  }

  switch(type) {
    case 'titulo':
      group = createCompGroup(id, type, cx - 150, cy - 20, 300, 50, '#cba6f7', 'Titulo', {texto: 'Minha Tela'});
      break;
    case 'lista':
      group = createCompGroup(id, type, cx - 200, cy - 80, 400, 180, '#89b4fa', 'Lista / Tabela', {modelo: 'produto', campos: 'nome, preco, status'});
      break;
    case 'botao':
      group = createCompGroup(id, type, cx - 60, cy - 18, 140, 40, '#a6e3a1', 'Botao', {texto: 'Novo', cor: 'azul', acao: 'criar produto'});
      break;
    case 'busca':
      group = createCompGroup(id, type, cx - 150, cy - 18, 300, 40, '#94e2d5', 'Busca', {modelo: 'produto'});
      break;
    case 'grafico':
      group = createCompGroup(id, type, cx - 180, cy - 80, 360, 180, '#f9e2af', 'Grafico', {modelo: 'produto', tipo: 'barra'});
      break;
    case 'cards':
      group = createCompGroup(id, type, cx - 200, cy - 50, 400, 100, '#cba6f7', 'Dashboard Cards', {});
      break;
    case 'formulario':
      group = createCompGroup(id, type, cx - 160, cy - 100, 320, 220, '#f5c2e7', 'Formulario', {modelo: 'produto'});
      break;
    case 'texto':
      group = createCompGroup(id, type, cx - 100, cy - 15, 200, 35, '#a6adc8', 'Texto', {conteudo: 'Texto aqui...'});
      break;
    case 'imagem':
      group = createCompGroup(id, type, cx - 80, cy - 80, 160, 160, '#94e2d5', 'Imagem', {url: ''});
      break;
    case 'separador':
      group = createCompGroup(id, type, cx - 150, cy - 2, 300, 4, '#585b70', 'Separador', {});
      break;
    case 'input':
      group = createCompGroup(id, type, cx - 120, cy - 18, 240, 40, '#b4befe', 'Campo', {nome: 'campo', tipo: 'texto'});
      break;
    case 'select':
      group = createCompGroup(id, type, cx - 120, cy - 18, 240, 40, '#fab387', 'Dropdown', {nome: 'campo', opcoes: 'opcao1, opcao2, opcao3'});
      break;
  }

  if (group) {
    fabricCanvas.add(group);
    fabricCanvas.setActiveObject(group);
    fabricCanvas.renderAll();
    showDesignerProps(group);
  }
}

function createCompGroup(id, type, x, y, w, h, color, label, props) {
  // Card background with colored left border effect
  var bg = new fabric.Rect({
    width: w, height: h,
    fill: '#181825',
    stroke: '#313244',
    strokeWidth: 1,
    rx: 8, ry: 8,
    originX: 'left', originY: 'top',
  });

  // Colored left accent bar
  var accent = new fabric.Rect({
    width: 3, height: h - 8,
    fill: color,
    rx: 1.5, ry: 1.5,
    left: 4, top: 4,
    originX: 'left', originY: 'top',
  });

  var labelText = new fabric.Text(label, {
    fontSize: 10,
    fill: color,
    fontFamily: 'Inter, system-ui, sans-serif',
    fontWeight: '600',
    left: 14,
    top: 6,
    originX: 'left', originY: 'top',
  });

  var previewText = '';
  switch(type) {
    case 'titulo': previewText = props.texto || 'Titulo'; break;
    case 'lista': previewText = 'ID | ' + (props.campos||'nome, preco').split(',').join(' | '); break;
    case 'botao': previewText = '[ ' + (props.texto||'Botao') + ' ]'; break;
    case 'busca': previewText = 'Buscar em ' + (props.modelo||'...'); break;
    case 'grafico': previewText = (props.tipo||'barra') + ' - ' + (props.modelo||'dados'); break;
    case 'cards': previewText = '[ Card 1 ] [ Card 2 ] [ Card 3 ]'; break;
    case 'formulario': previewText = 'Nome: [____]  Email: [____]  [ Salvar ]'; break;
    case 'texto': previewText = props.conteudo || 'Texto'; break;
    case 'imagem': previewText = '[ Imagem ]'; break;
    case 'separador': previewText = ''; break;
    case 'input': previewText = (props.nome||'campo') + ': [____________]'; break;
    case 'select': previewText = (props.nome||'campo') + ': [v selecione ]'; break;
  }

  var content = new fabric.Text(previewText, {
    fontSize: 12,
    fill: '#a6adc8',
    fontFamily: "'JetBrains Mono', monospace",
    left: 14,
    top: 22,
    originX: 'left', originY: 'top',
  });

  var group = new fabric.Group([bg, accent, labelText, content], {
    left: x, top: y,
    originX: 'left', originY: 'top',
    cornerStyle: 'circle',
    cornerColor: '#89b4fa',
    cornerStrokeColor: '#1e1e2e',
    cornerSize: 8,
    transparentCorners: false,
    borderColor: '#89b4fa',
    borderDashArray: [4, 4],
    borderScaleFactor: 1.5,
    padding: 4,
    snapAngle: 45,
  });

  group.compId = id;
  group.compType = type;
  group.compProps = props;
  group.compColor = color;
  group.compLabel = label;

  return group;
}

function showDesignerProps(obj) {
  if (!obj || !obj.compType) { clearDesignerProps(); return; }
  var panel = document.getElementById('d-props');
  var type = obj.compType;
  var props = obj.compProps || {};
  var color = obj.compColor || '#89b4fa';

  var fields = {
    titulo: [{key:'texto',label:'Texto',type:'text'}],
    lista: [{key:'modelo',label:'Modelo',type:'text'},{key:'campos',label:'Campos (virgula)',type:'text'}],
    botao: [{key:'texto',label:'Texto',type:'text'},{key:'cor',label:'Cor',type:'select',options:['azul','verde','vermelho','amarelo','roxo']},{key:'acao',label:'Acao',type:'text'}],
    busca: [{key:'modelo',label:'Modelo',type:'text'}],
    grafico: [{key:'modelo',label:'Modelo',type:'text'},{key:'tipo',label:'Tipo',type:'select',options:['barra','pizza','linha','doughnut']}],
    cards: [],
    formulario: [{key:'modelo',label:'Modelo',type:'text'}],
    texto: [{key:'conteudo',label:'Conteudo',type:'textarea'}],
    imagem: [{key:'url',label:'URL da imagem',type:'text'}],
    separador: [],
    input: [{key:'nome',label:'Nome do campo',type:'text'},{key:'tipo',label:'Tipo',type:'select',options:['texto','numero','email','telefone','data','senha','dinheiro']}],
    select: [{key:'nome',label:'Nome do campo',type:'text'},{key:'opcoes',label:'Opcoes (virgula)',type:'text'}]
  };

  var html = '<div>';
  // Component header with color dot
  html += '<div style="display:flex;align-items:center;gap:8px;margin-bottom:14px;padding-bottom:10px;border-bottom:1px solid var(--surface0)">';
  html += '<div style="width:10px;height:10px;border-radius:50%;background:'+color+';flex-shrink:0"></div>';
  html += '<span style="font-size:12px;font-weight:600;color:'+color+'">'+obj.compLabel+'</span>';
  html += '</div>';

  // Property fields
  (fields[type]||[]).forEach(function(f) {
    html += '<div class="prop-field" style="margin-bottom:10px">';
    html += '<label>'+f.label+'</label>';
    if (f.type === 'select') {
      html += '<select onchange="updateCanvasProp(\''+f.key+'\',this.value)" class="prop-input">';
      f.options.forEach(function(o) {
        html += '<option value="'+o+'"'+(props[f.key]===o?' selected':'')+'>'+o+'</option>';
      });
      html += '</select>';
    } else if (f.type === 'textarea') {
      html += '<textarea onchange="updateCanvasProp(\''+f.key+'\',this.value)" class="prop-input" rows="3" style="resize:none">'+(props[f.key]||'')+'</textarea>';
    } else {
      html += '<input type="text" value="'+(props[f.key]||'')+'" onchange="updateCanvasProp(\''+f.key+'\',this.value)" class="prop-input">';
    }
    html += '</div>';
  });

  // Position & Size section
  html += '<div style="margin-top:14px;padding-top:12px;border-top:1px solid var(--surface0)">';
  html += '<div style="font-size:10px;font-weight:600;color:var(--overlay0);text-transform:uppercase;letter-spacing:0.5px;margin-bottom:8px">Posicao e Tamanho</div>';
  html += '<div class="pos-grid">';
  html += '<div class="pos-cell"><span class="pos-label">X</span><span class="pos-val">'+Math.round(obj.left)+'</span></div>';
  html += '<div class="pos-cell"><span class="pos-label">Y</span><span class="pos-val">'+Math.round(obj.top)+'</span></div>';
  html += '<div class="pos-cell"><span class="pos-label">W</span><span class="pos-val">'+Math.round(obj.width * obj.scaleX)+'</span></div>';
  html += '<div class="pos-cell"><span class="pos-label">H</span><span class="pos-val">'+Math.round(obj.height * obj.scaleY)+'</span></div>';
  html += '</div></div>';

  // Actions
  html += '<div style="margin-top:14px;display:flex;gap:6px">';
  html += '<button onclick="duplicateSelected()" class="prop-action-btn" style="flex:1">Duplicar</button>';
  html += '<button onclick="deleteSelected()" class="prop-action-btn danger" style="flex:1">Remover</button>';
  html += '</div>';

  html += '</div>';
  panel.innerHTML = html;
}

function clearDesignerProps() {
  document.getElementById('d-props').innerHTML = '<p style="font-size:11px;color:var(--overlay0)">Clique num componente no canvas</p>';
}

function updateCanvasProp(key, val) {
  var obj = fabricCanvas.getActiveObject();
  if (!obj || !obj.compProps) return;
  obj.compProps[key] = val;

  var items = obj.getObjects();
  if (items.length >= 4) {
    var previewText = '';
    switch(obj.compType) {
      case 'titulo': previewText = obj.compProps.texto || 'Titulo'; break;
      case 'lista': previewText = 'ID | ' + (obj.compProps.campos||'').split(',').join(' | '); break;
      case 'botao': previewText = '[ ' + (obj.compProps.texto||'Botao') + ' ]'; break;
      case 'busca': previewText = 'Buscar em ' + (obj.compProps.modelo||'...'); break;
      case 'grafico': previewText = (obj.compProps.tipo||'barra') + ' - ' + (obj.compProps.modelo||'dados'); break;
      case 'formulario': previewText = 'Formulario: ' + (obj.compProps.modelo||'item'); break;
      case 'texto': previewText = obj.compProps.conteudo || 'Texto'; break;
      case 'input': previewText = (obj.compProps.nome||'campo') + ': [____________]'; break;
      case 'select': previewText = (obj.compProps.nome||'campo') + ': [v selecione ]'; break;
      default: previewText = obj.compLabel;
    }
    items[3].set('text', previewText);
  }
  fabricCanvas.renderAll();
  updateDesignerCode();
}

function deleteSelected() {
  var obj = fabricCanvas.getActiveObject();
  if (obj) {
    fabricCanvas.remove(obj);
    fabricCanvas.discardActiveObject();
    clearDesignerProps();
    updateDesignerCode();
  }
}

function duplicateSelected() {
  var obj = fabricCanvas.getActiveObject();
  if (!obj || !obj.compType) return;
  addToCanvas(obj.compType, obj.left + 20, obj.top + 20);
}

function clearCanvas() {
  if (!fabricCanvas) return;
  var objects = fabricCanvas.getObjects().filter(function(o){ return o.compType; });
  objects.forEach(function(o){ fabricCanvas.remove(o); });
  fabricCanvas.discardActiveObject();
  clearDesignerProps();
  updateDesignerCode();
}

function zoomIn() {
  if (!fabricCanvas) return;
  var z = fabricCanvas.getZoom() * 1.2;
  if (z > 5) z = 5;
  fabricCanvas.setZoom(z);
  document.getElementById('zoom-level').textContent = Math.round(z*100)+'%';
}
function zoomOut() {
  if (!fabricCanvas) return;
  var z = fabricCanvas.getZoom() * 0.8;
  if (z < 0.2) z = 0.2;
  fabricCanvas.setZoom(z);
  document.getElementById('zoom-level').textContent = Math.round(z*100)+'%';
}
function zoomReset() {
  if (!fabricCanvas) return;
  fabricCanvas.setZoom(1);
  fabricCanvas.viewportTransform = [1,0,0,1,0,0];
  fabricCanvas.renderAll();
  document.getElementById('zoom-level').textContent = '100%';
}

function updateDesignerCode() {
  if (!fabricCanvas) return;
  var objects = fabricCanvas.getObjects().filter(function(o){ return o.compType; });
  objects.sort(function(a,b){ return a.top - b.top; });

  var screenName = document.getElementById('d-screen-name').value || 'principal';
  var code = 'telas\n\n';
  code += '  tela ' + screenName + '\n';

  objects.forEach(function(obj) {
    var p = obj.compProps || {};
    switch(obj.compType) {
      case 'titulo':
        code += '    titulo "' + (p.texto||'Titulo') + '"\n';
        break;
      case 'lista':
        code += '    lista ' + (p.modelo||'item') + '\n';
        (p.campos||'').split(',').forEach(function(f) {
          f = f.trim();
          if (f) code += '      mostrar ' + f + '\n';
        });
        break;
      case 'botao':
        code += '    botao ' + (p.cor||'azul') + '\n';
        code += '      texto "' + (p.texto||'Novo') + '"\n';
        break;
      case 'busca':
        code += '    busca ' + (p.modelo||'item') + '\n';
        break;
      case 'grafico':
        code += '    grafico ' + (p.modelo||'item') + '\n';
        code += '      tipo ' + (p.tipo||'barra') + '\n';
        break;
      case 'cards':
        code += '    dashboard\n';
        break;
      case 'formulario':
        code += '    formulario ' + (p.modelo||'item') + '\n';
        break;
      case 'texto':
        code += '    # ' + (p.conteudo||'Texto') + '\n';
        break;
      case 'imagem':
        if (p.url) code += '    # imagem: ' + p.url + '\n';
        break;
      case 'separador':
        code += '    # ---\n';
        break;
      case 'input':
        code += '    # campo ' + (p.nome||'campo') + ': ' + (p.tipo||'texto') + '\n';
        break;
      case 'select':
        code += '    # select ' + (p.nome||'campo') + ': ' + (p.opcoes||'') + '\n';
        break;
    }
  });

  code += '\n';
  document.getElementById('d-generated').textContent = code;
}

function generateFromDesigner() {
  updateDesignerCode();
  var code = document.getElementById('d-generated').textContent;
  if (!code || code.trim() === 'telas\n\n  tela principal') {
    termLog('error', 'Canvas vazio - arraste componentes primeiro');
    return;
  }
  var filename = 'telas_visual.ge';
  fetch('/api/file/save', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({path: filename, content: code})
  }).then(function() {
    termLog('success', 'Telas salvas em ' + filename);
    loadFileTree();
    openFile(filename);
    switchMode('editor');
  });
}

// ============================================================
// FLUXOS - Flow Logic Editor
// ============================================================

var flowNodes = [];
var flowConnections = [];
var flowNodeId = 0;
var selectedNode = null;
var draggingNode = null;
var dragOffset = {x:0, y:0};
var connecting = false;
var connectFrom = null;

var nodeTypes = {
  trigger_click: {label:'Quando Clicar', color:'#a6e3a1', category:'trigger', fields:[{key:'botao',label:'Botao',type:'text'}]},
  trigger_create: {label:'Quando Criar', color:'#a6e3a1', category:'trigger', fields:[{key:'modelo',label:'Modelo',type:'text'}]},
  trigger_update: {label:'Quando Atualizar', color:'#a6e3a1', category:'trigger', fields:[{key:'modelo',label:'Modelo',type:'text'}]},
  trigger_cron: {label:'Agendamento', color:'#a6e3a1', category:'trigger', fields:[{key:'intervalo',label:'Intervalo',type:'text'},{key:'unidade',label:'Unidade',type:'select',options:['minutos','horas']}]},
  action_create: {label:'Criar Registro', color:'#89b4fa', category:'action', fields:[{key:'modelo',label:'Modelo',type:'text'}]},
  action_update: {label:'Atualizar Registro', color:'#89b4fa', category:'action', fields:[{key:'modelo',label:'Modelo',type:'text'}]},
  action_delete: {label:'Deletar Registro', color:'#89b4fa', category:'action', fields:[{key:'modelo',label:'Modelo',type:'text'}]},
  action_message: {label:'Mostrar Mensagem', color:'#89b4fa', category:'action', fields:[{key:'mensagem',label:'Mensagem',type:'text'}]},
  integ_whatsapp: {label:'Enviar WhatsApp', color:'#a6e3a1', category:'integration', fields:[{key:'para',label:'Para (campo)',type:'text'},{key:'mensagem',label:'Mensagem',type:'text'}]},
  integ_email: {label:'Enviar Email', color:'#fab387', category:'integration', fields:[{key:'para',label:'Para (campo)',type:'text'},{key:'assunto',label:'Assunto',type:'text'},{key:'corpo',label:'Corpo',type:'text'}]},
  integ_http: {label:'Chamar API', color:'#94e2d5', category:'integration', fields:[{key:'url',label:'URL',type:'text'},{key:'metodo',label:'Metodo',type:'select',options:['GET','POST','PUT','DELETE']}]},
  logic_if: {label:'Condicao (Se)', color:'#f9e2af', category:'logic', fields:[{key:'campo',label:'Campo',type:'text'},{key:'operador',label:'Operador',type:'select',options:['igual','maior','menor','diferente']},{key:'valor',label:'Valor',type:'text'}]},
  logic_loop: {label:'Para Cada', color:'#f9e2af', category:'logic', fields:[{key:'variavel',label:'Variavel',type:'text'},{key:'colecao',label:'Colecao',type:'text'}]},
  logic_set: {label:'Definir Variavel', color:'#f9e2af', category:'logic', fields:[{key:'nome',label:'Nome',type:'text'},{key:'valor',label:'Valor',type:'text'}]},
  logic_function: {label:'Funcao', color:'#f9e2af', category:'logic', fields:[{key:'nome',label:'Nome',type:'text'},{key:'params',label:'Parametros',type:'text'}]}
};

function dragNode(e, type) {
  e.dataTransfer.setData('nodeType', type);
}

function dropNode(e) {
  e.preventDefault();
  var type = e.dataTransfer.getData('nodeType');
  if (!type || !nodeTypes[type]) return;

  var rect = document.getElementById('flow-canvas').getBoundingClientRect();
  var x = e.clientX - rect.left - 80;
  var y = e.clientY - rect.top - 30;

  var node = {id: ++flowNodeId, type: type, x: x, y: y, props: {}};
  nodeTypes[type].fields.forEach(function(f) { node.props[f.key] = ''; });
  flowNodes.push(node);
  renderFlowNodes();
}

function renderFlowNodes() {
  var container = document.getElementById('flow-nodes');
  container.innerHTML = '';

  flowNodes.forEach(function(node) {
    var nt = nodeTypes[node.type];
    var div = document.createElement('div');
    div.className = 'flow-node' + (node === selectedNode ? ' selected' : '');
    div.style.left = node.x + 'px';
    div.style.top = node.y + 'px';
    div.setAttribute('data-id', node.id);

    var bodyText = '';
    if (nt.fields.length > 0) {
      bodyText = node.props[nt.fields[0].key] || nt.label;
    } else {
      bodyText = nt.label;
    }

    div.innerHTML = '<div class="node-head" style="background:'+nt.color+'12;color:'+nt.color+'">'+
      '<span>'+nt.label+'</span>'+
      '<button onclick="event.stopPropagation();removeNode('+node.id+')" style="background:none;border:none;color:'+nt.color+';cursor:pointer;font-size:14px;opacity:0.5">&times;</button>'+
      '</div>'+
      '<div class="node-body">'+bodyText+'</div>'+
      '<div class="flow-port port-in" onmousedown="event.stopPropagation();startConnect('+node.id+',\'in\',event)" data-node="'+node.id+'" data-port="in"></div>'+
      '<div class="flow-port port-out" onmousedown="event.stopPropagation();startConnect('+node.id+',\'out\',event)" data-node="'+node.id+'" data-port="out"></div>';

    div.onmousedown = function(e) {
      if (e.target.classList.contains('flow-port')) return;
      selectedNode = node;
      draggingNode = node;
      var r = div.getBoundingClientRect();
      dragOffset = {x: e.clientX - r.left, y: e.clientY - r.top};
      renderFlowNodes();
      renderFlowProps();
    };

    container.appendChild(div);
  });

  renderFlowConnections();
}

function flowCanvasMouseDown(e) {
  if (e.target.id === 'flow-canvas' || (e.target.closest('#flow-canvas') && !e.target.closest('.flow-node'))) {
    selectedNode = null;
    renderFlowNodes();
    renderFlowProps();
  }
}

function flowCanvasMouseMove(e) {
  if (draggingNode) {
    var rect = document.getElementById('flow-canvas').getBoundingClientRect();
    draggingNode.x = e.clientX - rect.left - dragOffset.x;
    draggingNode.y = e.clientY - rect.top - dragOffset.y;
    var el = document.querySelector('[data-id="'+draggingNode.id+'"]');
    if (el) {
      el.style.left = draggingNode.x + 'px';
      el.style.top = draggingNode.y + 'px';
    }
    renderFlowConnections();
  }
}

function flowCanvasMouseUp(e) {
  if (connecting && connectFrom) {
    var port = e.target.closest('.flow-port');
    if (port) {
      var toId = parseInt(port.getAttribute('data-node'));
      var toPort = port.getAttribute('data-port');
      if (connectFrom.id !== toId && connectFrom.port !== toPort) {
        var fromId = connectFrom.port === 'out' ? connectFrom.id : toId;
        var targetId = connectFrom.port === 'out' ? toId : connectFrom.id;
        var exists = flowConnections.some(function(c){return c.from===fromId && c.to===targetId;});
        if (!exists) {
          flowConnections.push({from: fromId, to: targetId});
          renderFlowConnections();
          updateFlowCode();
        }
      }
    }
    connecting = false;
    connectFrom = null;
  }
  draggingNode = null;
}

function startConnect(nodeId, port, e) {
  connecting = true;
  connectFrom = {id: nodeId, port: port};
}

function renderFlowConnections() {
  var svg = document.getElementById('flow-svg');
  svg.innerHTML = '';

  flowConnections.forEach(function(conn) {
    var fromEl = document.querySelector('[data-id="'+conn.from+'"] .port-out');
    var toEl = document.querySelector('[data-id="'+conn.to+'"] .port-in');
    if (!fromEl || !toEl) return;

    var canvas = document.getElementById('flow-canvas').getBoundingClientRect();
    var fromRect = fromEl.getBoundingClientRect();
    var toRect = toEl.getBoundingClientRect();

    var x1 = fromRect.left - canvas.left + 6;
    var y1 = fromRect.top - canvas.top + 6;
    var x2 = toRect.left - canvas.left + 6;
    var y2 = toRect.top - canvas.top + 6;

    var dx = Math.abs(x2 - x1) * 0.5;
    var path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path.setAttribute('d', 'M'+x1+','+y1+' C'+(x1+dx)+','+y1+' '+(x2-dx)+','+y2+' '+x2+','+y2);
    path.setAttribute('stroke', '#89b4fa');
    path.setAttribute('stroke-width', '2');
    path.setAttribute('fill', 'none');
    path.setAttribute('stroke-dasharray', '');
    path.style.pointerEvents = 'stroke';
    path.onclick = function() {
      flowConnections = flowConnections.filter(function(c){return c !== conn;});
      renderFlowConnections();
      updateFlowCode();
    };
    path.style.cursor = 'pointer';
    path.onmouseenter = function(){this.setAttribute('stroke','#f38ba8');this.setAttribute('stroke-width','3');};
    path.onmouseleave = function(){this.setAttribute('stroke','#89b4fa');this.setAttribute('stroke-width','2');};
    svg.appendChild(path);
  });
}

function removeNode(id) {
  flowNodes = flowNodes.filter(function(n){return n.id !== id;});
  flowConnections = flowConnections.filter(function(c){return c.from !== id && c.to !== id;});
  if (selectedNode && selectedNode.id === id) selectedNode = null;
  renderFlowNodes();
  renderFlowProps();
  updateFlowCode();
}

function renderFlowProps() {
  var panel = document.getElementById('flow-props');
  if (!selectedNode) {
    panel.innerHTML = '<p style="font-size:11px;color:var(--overlay0)">Selecione um nodo</p>';
    return;
  }
  var nt = nodeTypes[selectedNode.type];
  var html = '<div>';
  html += '<div style="display:flex;align-items:center;gap:8px;margin-bottom:14px;padding-bottom:10px;border-bottom:1px solid var(--surface0)">';
  html += '<div style="width:10px;height:10px;border-radius:50%;background:'+nt.color+';flex-shrink:0"></div>';
  html += '<span style="font-size:12px;font-weight:600;color:'+nt.color+'">'+nt.label+'</span>';
  html += '</div>';

  nt.fields.forEach(function(f) {
    html += '<div class="prop-field" style="margin-bottom:10px">';
    html += '<label>'+f.label+'</label>';
    if (f.type === 'select') {
      html += '<select onchange="updateNodeProp(\''+f.key+'\',this.value)" class="prop-input">';
      f.options.forEach(function(o) {
        html += '<option value="'+o+'"'+(selectedNode.props[f.key]===o?' selected':'')+'>'+o+'</option>';
      });
      html += '</select>';
    } else {
      html += '<input type="text" value="'+(selectedNode.props[f.key]||'')+'" onchange="updateNodeProp(\''+f.key+'\',this.value)" class="prop-input">';
    }
    html += '</div>';
  });
  html += '</div>';
  panel.innerHTML = html;
}

function updateNodeProp(key, val) {
  if (!selectedNode) return;
  selectedNode.props[key] = val;
  renderFlowNodes();
  updateFlowCode();
}

function updateFlowCode() {
  var code = '';
  var triggers = flowNodes.filter(function(n){return nodeTypes[n.type].category==='trigger';});

  if (triggers.length > 0) {
    code += 'eventos\n\n';
    triggers.forEach(function(trigger) {
      switch(trigger.type) {
        case 'trigger_click':
          code += '  quando clicar "' + (trigger.props.botao||'Botao') + '"\n';
          break;
        case 'trigger_create':
          code += '  quando criar ' + (trigger.props.modelo||'item') + '\n';
          break;
        case 'trigger_update':
          code += '  quando atualizar ' + (trigger.props.modelo||'item') + '\n';
          break;
        case 'trigger_cron':
          code += '  cada ' + (trigger.props.intervalo||'5') + ' ' + (trigger.props.unidade||'minutos') + '\n';
          break;
      }
      var connected = getConnectedNodes(trigger.id);
      connected.forEach(function(node) {
        code += '    ' + nodeToCode(node) + '\n';
      });
      code += '\n';
    });
  }

  var logicNodes = flowNodes.filter(function(n){
    return nodeTypes[n.type].category==='logic' && !flowConnections.some(function(c){return c.to===n.id;});
  });
  if (logicNodes.length > 0) {
    code += 'logica\n\n';
    logicNodes.forEach(function(n) {
      code += '  ' + nodeToCode(n) + '\n';
      var connected = getConnectedNodes(n.id);
      connected.forEach(function(cn) {
        code += '    ' + nodeToCode(cn) + '\n';
      });
    });
  }

  document.getElementById('flow-generated-code').textContent = code;
}

function getConnectedNodes(fromId) {
  var result = [];
  flowConnections.filter(function(c){return c.from===fromId;}).forEach(function(c) {
    var node = flowNodes.find(function(n){return n.id===c.to;});
    if (node) result.push(node);
  });
  return result;
}

function nodeToCode(node) {
  switch(node.type) {
    case 'action_create': return 'criar ' + (node.props.modelo||'item');
    case 'action_update': return 'atualizar ' + (node.props.modelo||'item');
    case 'action_delete': return 'deletar ' + (node.props.modelo||'item');
    case 'action_message': return 'mostrar "' + (node.props.mensagem||'') + '"';
    case 'integ_whatsapp': return 'enviar mensagem para ' + (node.props.para||'telefone') + ' texto "' + (node.props.mensagem||'') + '"';
    case 'integ_email': return 'enviar email para ' + (node.props.para||'email') + ' assunto "' + (node.props.assunto||'') + '"';
    case 'integ_http': return 'chamar api "' + (node.props.url||'') + '"';
    case 'logic_if': return 'se ' + (node.props.campo||'x') + ' ' + (node.props.operador||'igual') + ' ' + (node.props.valor||'0');
    case 'logic_loop': return 'para cada ' + (node.props.variavel||'item') + ' em ' + (node.props.colecao||'lista');
    case 'logic_set': return 'definir ' + (node.props.nome||'x') + ' = ' + (node.props.valor||'0');
    case 'logic_function': return 'funcao ' + (node.props.nome||'minha_funcao') + '(' + (node.props.params||'') + ')';
    default: return '# ' + node.type;
  }
}

function generateFromFlow() {
  updateFlowCode();
  var code = document.getElementById('flow-generated-code').textContent;
  if (!code) { termLog('error', 'Nenhum fluxo criado'); return; }

  fetch('/api/file/save', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({path: 'fluxo_visual.ge', content: code})
  }).then(function() {
    termLog('success', 'Fluxo salvo em fluxo_visual.ge');
    loadFileTree();
    openFile('fluxo_visual.ge');
    switchMode('editor');
  });
}

function clearFlow() {
  flowNodes = [];
  flowConnections = [];
  selectedNode = null;
  renderFlowNodes();
  renderFlowProps();
  document.getElementById('flow-generated-code').textContent = '';
}

// ============================================================
// LIVE PREVIEW
// ============================================================

var previewOpen = false;
function togglePreview() {
  previewOpen = !previewOpen;
  var panel = document.getElementById('preview-panel');
  if (previewOpen) {
    panel.style.display = 'block';
    panel.style.height = '45%';
    document.getElementById('preview-iframe').src = 'http://localhost:8080';
    document.getElementById('btn-preview').classList.add('active');
  } else {
    panel.style.display = 'none';
    document.getElementById('preview-iframe').src = '';
    document.getElementById('btn-preview').classList.remove('active');
  }
}
function refreshPreview() {
  var iframe = document.getElementById('preview-iframe');
  iframe.src = iframe.src;
}

// ============================================================
// UNDO/REDO FOR CANVAS
// ============================================================

var canvasHistory = [];
var canvasHistoryIndex = -1;
var canvasIgnoreChange = false;

function saveCanvasState() {
  if (canvasIgnoreChange || !fabricCanvas) return;
  var state = JSON.stringify(fabricCanvas.toJSON(['compId','compType','compProps','compColor','compLabel']));
  // Remove future states if we undid
  canvasHistory = canvasHistory.slice(0, canvasHistoryIndex + 1);
  canvasHistory.push(state);
  if (canvasHistory.length > 50) canvasHistory.shift(); // limit
  canvasHistoryIndex = canvasHistory.length - 1;
}

function undoCanvas() {
  if (canvasHistoryIndex <= 0 || !fabricCanvas) return;
  canvasHistoryIndex--;
  canvasIgnoreChange = true;
  fabricCanvas.loadFromJSON(canvasHistory[canvasHistoryIndex], function() {
    fabricCanvas.renderAll();
    canvasIgnoreChange = false;
    updateDesignerCode();
    termLog('info', 'Desfazer');
  });
}

function redoCanvas() {
  if (canvasHistoryIndex >= canvasHistory.length - 1 || !fabricCanvas) return;
  canvasHistoryIndex++;
  canvasIgnoreChange = true;
  fabricCanvas.loadFromJSON(canvasHistory[canvasHistoryIndex], function() {
    fabricCanvas.renderAll();
    canvasIgnoreChange = false;
    updateDesignerCode();
    termLog('info', 'Refazer');
  });
}
</script>
</body>
</html>`
