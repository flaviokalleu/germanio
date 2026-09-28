#!/usr/bin/env python3
"""Generates themes/germanio-{escuro,claro}-color-theme.json.

The palette follows the logo (violet → teal). Each role of a Germanio line
has its own colour: intent words, permissions, data, roles, actions, fields,
modifiers, types and states.
"""
import json, os

ROLES = {
    #                          dark       light
    "keyword.other.intent":     ("#c4a7ff", "#6a3fc8", "bold"),
    "keyword.other.permission": ("#4fd8c7", "#0b7d73", "bold"),
    "keyword.other.event":      ("#ff9fd4", "#b0306f", "bold"),
    "keyword.other.block":      ("#c4a7ff", "#6a3fc8", "bold"),
    "keyword.control":          ("#ff8fb8", "#c0266a", ""),
    "keyword.operator":         ("#9aa7c7", "#5a6682", ""),
    "entity.name.type":         ("#7cc7ff", "#1565c0", ""),
    "entity.name.section":      ("#ffffff", "#1a1530", "bold"),
    "entity.name.class.role":   ("#ffc07a", "#b35900", ""),
    "entity.name.function.action": ("#8fe39b", "#1b7a2c", ""),
    "entity.name.function":     ("#82b8ff", "#1f5fbf", ""),
    "variable.other.property":  ("#e8e4f7", "#2c2540", ""),
    "storage.modifier":         ("#f0a6e8", "#9b2a8c", "italic"),
    "storage.type.function":    ("#c4a7ff", "#6a3fc8", "bold"),
    "support.type":             ("#5fe0e8", "#007b85", ""),
    "constant.other.state":     ("#ffd77a", "#9a6b00", "bold"),
    "constant.language":        ("#ff9f6b", "#c1440e", ""),
    "constant.numeric":         ("#ff9f6b", "#c1440e", ""),
    "string":                   ("#a8e6a1", "#2e7d32", ""),
    "meta.interpolation":       ("#ffd77a", "#9a6b00", ""),
    "variable.language":        ("#e5b86b", "#8d5b00", "italic"),
    "support.class.module":     ("#5fe0e8", "#007b85", ""),
    "support.function":         ("#82b8ff", "#1f5fbf", ""),
    "comment":                  ("#7d7896", "#8a8599", "italic"),
    "constant.language.http-method": ("#ffd77a", "#9a6b00", "bold"),
}

UI = {
    "escuro": {"editor.background": "#17141f", "editor.foreground": "#e8e4f7", "editorLineNumber.foreground": "#4f4867",
               "editorLineNumber.activeForeground": "#b8a8e8", "editor.selectionBackground": "#4b3a8080",
               "editor.lineHighlightBackground": "#211c2e", "editorCursor.foreground": "#4fd8c7",
               "editorIndentGuide.background1": "#2c2640", "sideBar.background": "#13111a", "activityBar.background": "#13111a",
               "titleBar.activeBackground": "#13111a", "statusBar.background": "#5a37b8", "statusBar.foreground": "#ffffff",
               "tab.activeBackground": "#17141f", "tab.inactiveBackground": "#13111a", "editorGroupHeader.tabsBackground": "#13111a",
               "focusBorder": "#7c4ddb", "button.background": "#6a3fc8", "badge.background": "#0f9b9b"},
    "claro": {"editor.background": "#fcfbff", "editor.foreground": "#2c2540", "editorLineNumber.foreground": "#b6b0c8",
              "editorLineNumber.activeForeground": "#5a37b8", "editor.selectionBackground": "#c9b8f580",
              "editor.lineHighlightBackground": "#f3f0fb", "editorCursor.foreground": "#0b7d73",
              "sideBar.background": "#f5f3fb", "activityBar.background": "#f0edf9", "titleBar.activeBackground": "#f0edf9",
              "statusBar.background": "#5a37b8", "statusBar.foreground": "#ffffff", "focusBorder": "#7c4ddb",
              "button.background": "#6a3fc8", "badge.background": "#0f9b9b"},
}

here = os.path.dirname(os.path.abspath(__file__))
for variant, idx, kind in (("escuro", 0, "dark"), ("claro", 1, "light")):
    rules = []
    for scope, (dark, light, style) in ROLES.items():
        s = {"foreground": (dark, light)[idx]}
        if style:
            s["fontStyle"] = style
        rules.append({"name": scope, "scope": scope + ".germanio", "settings": s})
    theme = {"name": "Germanio " + variant.capitalize(), "type": kind, "colors": UI[variant],
             "semanticHighlighting": False, "tokenColors": rules}
    with open(os.path.join(here, "..", "themes", f"germanio-{variant}-color-theme.json"), "w", encoding="utf-8") as f:
        json.dump(theme, f, ensure_ascii=False, indent=2)
        f.write("\n")
print("ok")
