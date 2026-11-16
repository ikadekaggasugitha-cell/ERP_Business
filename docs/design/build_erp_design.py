#!/usr/bin/env python3
"""
High-craft FigCraft generator for Enterprise ERP Backoffice SPA (CONS-013).
Produces an 8-board studio file:
  Page 1 — 01 Frontend UI (Backoffice SPA): 3x2 grid of 1440x940 artboards
    01 Dashboard Eksekutif (Light)
    02 Buku Besar & Jurnal — Daftar & Filter
    03 SideDrawer (XL) — Detail Jurnal
    04 AppDialog (2XL) — Buat Entri Jurnal
    05 Dashboard Eksekutif (Dark Mode · html.dark)
    06 Responsif Mobile (390px) & Overlay Interaktif
  Page 2 — 02 Design System & UI Kit: 2x 1480x1040 boards
    01 Design Tokens, Variables & Brand
    02 UI Component Library & Interactive States
"""

from __future__ import annotations

import argparse
import json
import math
import os
import re
import subprocess
import tempfile
from pathlib import Path
from typing import Any

WORKSPACE_ROOT = Path("/Users/aggasugitha/Documents/erp.business")
DEFAULT_OUT = WORKSPACE_ROOT / "docs/design/erp-backoffice-spa.figcraft"
BRAND_LOGO_PNG = WORKSPACE_ROOT / "apps/web-spa/public/brand-logo.png"

# ==============================================================================
# 1. AUTHORITATIVE DESIGN TOKENS (apps/web-spa/src/index.css + Design Taste)
# ==============================================================================

BRAND_SCALE = {
    "50": "#EFF6FF",
    "100": "#DBEAFE",
    "200": "#BFDBFE",
    "300": "#93C5FD",
    "400": "#60A5FA",
    "500": "#3B82F6",
    "600": "#2563EB",
    "700": "#1D4ED8",
    "800": "#1E40AF",
    "900": "#1E3A8A",
    "950": "#172554",
}

SLATE_SCALE = {
    "50": "#F8FAFC",
    "100": "#F1F5F9",
    "200": "#E2E8F0",
    "300": "#CBD5E1",
    "400": "#94A3B8",
    "500": "#64748B",
    "600": "#475569",
    "700": "#334155",
    "800": "#1E293B",
    "900": "#0F172A",
    "950": "#020617",
}

ACCENT_SCALE = {
    "500": "#F97316",
    "600": "#EA580C",
}

CHART_COLORS = {
    "Asset": "#0284C7",
    "Liability": "#D97706",
    "Equity": "#4338CA",
    "Revenue": "#0F766E",
    "Expense": "#BE123C",
}

CHART_COLORS_DARK = {
    "Asset": "#38BDF8",
    "Liability": "#FBBF24",
    "Equity": "#818CF8",
    "Revenue": "#2DD4BF",
    "Expense": "#FB7185",
}

LIGHT = {
    "name": "Light",
    "canvas": "#F8FAFC",
    "surface": "#FFFFFF",
    "surface_alt": "#F8FAFC",
    "surface_muted": "#F1F5F9",
    "surface_hover": "#F8FAFC",
    "surface_selected": "#EFF6FF",
    "navbar": "#FFFFFF",
    "sidebar": "#FFFFFF",
    "border": "#E2E8F0",
    "border_subtle": "#F1F5F9",
    "border_strong": "#CBD5E1",
    "ink": "#0F172A",
    "ink_secondary": "#334155",
    "ink_muted": "#64748B",
    "ink_subtle": "#94A3B8",
    "brand_primary": "#1D4ED8",
    "brand_primary_hover": "#1E40AF",
    "brand_soft": "#EFF6FF",
    "brand_soft_border": "#BFDBFE",
    "brand_soft_ink": "#1E3A8A",
    "gridline": "#E2E8F0",
    "scrim": "#0F172A5C",
    "tooltip_bg": "#0F172A",
    "tooltip_border": "#1E293B",
    "tooltip_ink": "#F8FAFC",
    "tooltip_muted": "#94A3B8",
    "badges": {
        "neutral": ("#F1F5F9", "#E2E8F0", "#64748B", "#334155"),
        "success": ("#ECFDF5", "#A7F3D0", "#10B981", "#065F46"),
        "warning": ("#FFFBEB", "#FDE68A", "#F59E0B", "#92400E"),
        "danger": ("#FFF1F2", "#FECDD3", "#F43F5E", "#9F1239"),
        "info": ("#EFF6FF", "#BFDBFE", "#3B82F6", "#1E40AF"),
        "brand": ("#EEF2FF", "#C7D2FE", "#6366F1", "#3730A3"),
    },
    "kpi_icon": {
        "brand": ("#EFF6FF", "#BFDBFE", "#1D4ED8"),
        "warning": ("#FFFBEB", "#FDE68A", "#D97706"),
        "success": ("#ECFDF5", "#A7F3D0", "#059669"),
        "sky": ("#F0F9FF", "#BAE6FD", "#0284C7"),
        "indigo": ("#EEF2FF", "#C7D2FE", "#4338CA"),
    },
    "shadow_card": {"color": "#0F172A0A", "offset": [0, 1], "radius": 3, "spread": 0},
    "shadow_popover": {"color": "#0F172A1C", "offset": [0, 12], "radius": 28, "spread": -6},
    "shadow_modal": {"color": "#0F172A38", "offset": [0, 22], "radius": 48, "spread": -10},
    "shadow_drawer": {"color": "#0F172A2E", "offset": [-10, 0], "radius": 36, "spread": 0},
}

DARK = {
    "name": "Dark",
    "canvas": "#020617",
    "surface": "#0F172A",
    "surface_alt": "#0B1222",
    "surface_muted": "#1E293B",
    "surface_hover": "#162033",
    "surface_selected": "#17255470",
    "navbar": "#0B1120",
    "sidebar": "#0B1120",
    "border": "#1E293B",
    "border_subtle": "#172033",
    "border_strong": "#334155",
    "ink": "#F8FAFC",
    "ink_secondary": "#CBD5E1",
    "ink_muted": "#94A3B8",
    "ink_subtle": "#64748B",
    "brand_primary": "#2563EB",
    "brand_primary_hover": "#3B82F6",
    "brand_soft": "#17255480",
    "brand_soft_border": "#1E3A8A",
    "brand_soft_ink": "#93C5FD",
    "gridline": "#1E293B",
    "scrim": "#02061799",
    "tooltip_bg": "#1E293B",
    "tooltip_border": "#334155",
    "tooltip_ink": "#F8FAFC",
    "tooltip_muted": "#94A3B8",
    "badges": {
        "neutral": ("#1E293B", "#334155", "#94A3B8", "#CBD5E1"),
        "success": ("#064E3B70", "#065F46", "#34D399", "#6EE7B7"),
        "warning": ("#451A0380", "#78350F", "#FBBF24", "#FDE68A"),
        "danger": ("#4C051980", "#881337", "#FB7185", "#FDA4AF"),
        "info": ("#17255490", "#1E40AF", "#60A5FA", "#93C5FD"),
        "brand": ("#312E8180", "#3730A3", "#818CF8", "#A5B4FC"),
    },
    "kpi_icon": {
        "brand": ("#17255490", "#1E3A8A", "#60A5FA"),
        "warning": ("#451A0385", "#78350F", "#FBBF24"),
        "success": ("#064E3B85", "#065F46", "#34D399"),
        "sky": ("#0C4A6E85", "#0369A1", "#38BDF8"),
        "indigo": ("#312E8185", "#3730A3", "#A5B4FC"),
    },
    "shadow_card": {"color": "#00000033", "offset": [0, 1], "radius": 3, "spread": 0},
    "shadow_popover": {"color": "#00000066", "offset": [0, 14], "radius": 32, "spread": -4},
    "shadow_modal": {"color": "#00000080", "offset": [0, 22], "radius": 48, "spread": -8},
    "shadow_drawer": {"color": "#00000073", "offset": [-10, 0], "radius": 36, "spread": 0},
}

# ==============================================================================
# 2. VECTOR ICON ENGINE (Lucide 24x24 ViewBox -> Scaled Clean SVG Path)
# ==============================================================================

_SVG_TOKEN_RE = re.compile(r"[A-Za-z]|[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?")


def svg_circle(cx: float, cy: float, r: float) -> str:
    return (
        f"M {cx + r:g} {cy:g} "
        f"A {r:g} {r:g} 0 1 0 {cx - r:g} {cy:g} "
        f"A {r:g} {r:g} 0 1 0 {cx + r:g} {cy:g} Z"
    )


def svg_rect(x: float, y: float, w: float, h: float, rx: float = 0.0) -> str:
    if rx <= 0:
        return f"M {x:g} {y:g} H {x + w:g} V {y + h:g} H {x:g} Z"
    r = min(rx, w / 2.0, h / 2.0)
    return (
        f"M {x + r:g} {y:g} "
        f"H {x + w - r:g} A {r:g} {r:g} 0 0 1 {x + w:g} {y + r:g} "
        f"V {y + h - r:g} A {r:g} {r:g} 0 0 1 {x + w - r:g} {y + h:g} "
        f"H {x + r:g} A {r:g} {r:g} 0 0 1 {x:g} {y + h - r:g} "
        f"V {y + r:g} A {r:g} {r:g} 0 0 1 {x + r:g} {y:g} Z"
    )


def scale_svg_path(path_24: str, target_size: float) -> str:
    """Scale an SVG path authored in a 24x24 viewBox cleanly to target_size."""
    s = target_size / 24.0
    tokens = _SVG_TOKEN_RE.findall(path_24)
    out: list[str] = []
    cmd = ""
    i = 0
    param_counts = {
        "M": 2, "m": 2, "L": 2, "l": 2, "T": 2, "t": 2,
        "H": 1, "h": 1, "V": 1, "v": 1,
        "C": 6, "c": 6, "S": 4, "s": 4, "Q": 4, "q": 4,
        "A": 7, "a": 7, "Z": 0, "z": 0,
    }

    def fmt(v: float) -> str:
        val = round(v, 3)
        return f"{val:.3f}".rstrip("0").rstrip(".") or "0"

    while i < len(tokens):
        tok = tokens[i]
        if tok.isalpha():
            cmd = tok
            out.append(cmd)
            i += 1
            if cmd in ("Z", "z"):
                continue
            continue
        arity = param_counts.get(cmd, 0)
        if arity == 0 or i + arity > len(tokens):
            break
        nums = [float(tokens[i + k]) for k in range(arity)]
        if cmd in ("A", "a"):
            out.extend([
                fmt(nums[0] * s),
                fmt(nums[1] * s),
                str(int(round(nums[2]))),
                str(int(round(nums[3]))),
                str(int(round(nums[4]))),
                fmt(nums[5] * s),
                fmt(nums[6] * s),
            ])
        else:
            out.extend(fmt(n * s) for n in nums)
        i += arity
        if cmd == "M":
            cmd = "L"
        elif cmd == "m":
            cmd = "l"
    return " ".join(out)


LUCIDE_24: dict[str, str] = {
    "layout-dashboard": " ".join([
        svg_rect(3, 3, 7, 9, 1.5),
        svg_rect(14, 3, 7, 5, 1.5),
        svg_rect(14, 12, 7, 9, 1.5),
        svg_rect(3, 16, 7, 5, 1.5),
    ]),
    "book-open": (
        "M 2 3 H 8 A 4 4 0 0 1 12 7 V 21 A 3 3 0 0 0 9 18 H 2 Z "
        "M 22 3 H 16 A 4 4 0 0 0 12 7 V 21 A 3 3 0 0 1 15 18 H 22 Z"
    ),
    "layers": (
        "M 12 2 L 2 7 L 12 12 L 22 7 Z "
        "M 2 17 L 12 22 L 22 17 "
        "M 2 12 L 12 17 L 22 12"
    ),
    "arrow-down-left": "M 17 7 L 7 17 M 17 17 H 7 V 7",
    "arrow-up-right": "M 7 17 L 17 7 M 7 7 H 17 V 17",
    "receipt": (
        "M 4 2 V 22 L 6.5 20 L 9 22 L 12 20 L 15 22 L 17.5 20 L 20 22 V 2 "
        "L 17.5 4 L 15 2 L 12 4 L 9 2 L 6.5 4 Z "
        "M 8 9 H 16 M 8 13 H 16 M 8 17 H 13"
    ),
    "landmark": (
        "M 3 22 H 21 M 6 18 V 11 M 10 18 V 11 M 14 18 V 11 M 18 18 V 11 "
        "M 12 2 L 20 7 H 4 Z"
    ),
    "coins": (
        svg_circle(8.5, 8.5, 5.5) + " "
        "M 18.09 10.37 A 5.5 5.5 0 1 1 10.37 18.09 "
        "M 8.5 6.5 V 10.5 M 15 14 L 17 16"
    ),
    "shopping-cart": (
        svg_circle(9, 20.5, 1.2) + " " + svg_circle(19, 20.5, 1.2) + " "
        "M 2 3 H 5 L 7.4 14.5 A 2 2 0 0 0 9.35 16 H 18.5 "
        "A 2 2 0 0 0 20.45 14.4 L 22 6.5 H 6"
    ),
    "package": (
        "M 16.5 9.4 L 7.55 4.24 "
        "M 21 16 V 8 A 2 2 0 0 0 20 6.27 L 13 2.27 A 2 2 0 0 0 11 2.27 "
        "L 4 6.27 A 2 2 0 0 0 3 8 V 16 A 2 2 0 0 0 4 17.73 L 11 21.73 "
        "A 2 2 0 0 0 13 21.73 L 20 17.73 A 2 2 0 0 0 21 16 Z "
        "M 3.3 7 L 12 12 L 20.7 7 M 12 22 V 12"
    ),
    "trending-up": "M 22 7 L 13.5 15.5 L 8.5 10.5 L 2 17 M 16 7 H 22 V 13",
    "factory": (
        "M 2 20 A 2 2 0 0 0 4 22 H 20 A 2 2 0 0 0 22 20 V 8 L 15 13 V 8 "
        "L 8 13 V 4 A 2 2 0 0 0 6 2 H 4 A 2 2 0 0 0 2 4 Z "
        "M 7 18 H 8.5 M 12 18 H 13.5 M 17 18 H 18.5"
    ),
    "users": (
        "M 16 21 V 19 A 4 4 0 0 0 12 15 H 6 A 4 4 0 0 0 2 19 V 21 "
        "M 22 21 V 19 A 4 4 0 0 0 19 15.2 M 16 3.3 A 4 4 0 0 1 16 10.7 "
        + svg_circle(9, 7, 3.8)
    ),
    "wallet": (
        "M 19 7 V 4.5 A 1.5 1.5 0 0 0 17.5 3 H 5 A 2 2 0 0 0 5 7 H 20 "
        "A 1 1 0 0 1 21 8 V 19 A 2 2 0 0 1 19 21 H 5 A 2 2 0 0 1 3 19 V 5 "
        "M 16 14 H 21"
    ),
    "shield-check": (
        "M 20 12.5 C 20 17.5 16.5 20.3 12 22 C 7.5 20.3 4 17.5 4 12.5 "
        "V 5.8 L 12 2.5 L 20 5.8 Z M 9 12 L 11 14 L 15.5 9.5"
    ),
    "database": (
        "M 4 6 A 8 3 0 1 0 20 6 A 8 3 0 1 0 4 6 "
        "M 4 6 V 18 A 8 3 0 0 0 20 18 V 6 "
        "M 4 12 A 8 3 0 0 0 20 12"
    ),
    "briefcase": svg_rect(2, 7, 20, 14, 2) + " M 16 21 V 4.5 A 1.5 1.5 0 0 0 14.5 3 H 9.5 A 1.5 1.5 0 0 0 8 4.5 V 21",
    "bar-chart-3": "M 3 3 V 19 A 2 2 0 0 0 5 21 H 21 M 18 17 V 9 M 13 17 V 5 M 8 17 V 13",
    "search": svg_circle(10.5, 10.5, 7) + " M 21 21 L 15.6 15.6",
    "building-2": (
        "M 6 21 V 4 A 1.5 1.5 0 0 1 7.5 2.5 H 16.5 A 1.5 1.5 0 0 1 18 4 V 21 Z "
        "M 6 11 H 3.5 A 1.5 1.5 0 0 0 2 12.5 V 21 H 6 "
        "M 18 9 H 20.5 A 1.5 1.5 0 0 1 22 10.5 V 21 H 18 "
        "M 10 6.5 H 14 M 10 10.5 H 14 M 10 14.5 H 14"
    ),
    "bell": (
        "M 6 8.5 A 6 6 0 0 1 18 8.5 C 18 14.5 20.5 16.5 20.5 16.5 H 3.5 "
        "C 3.5 16.5 6 14.5 6 8.5 M 10.2 20 A 2 2 0 0 0 13.8 20"
    ),
    "moon": "M 12 3 A 6.5 6.5 0 0 0 21 12.5 A 9 9 0 1 1 12 3 Z",
    "sun": (
        svg_circle(12, 12, 4.2) + " "
        "M 12 2 V 4.2 M 12 19.8 V 22 M 4.9 4.9 L 6.5 6.5 M 17.5 17.5 L 19.1 19.1 "
        "M 2 12 H 4.2 M 19.8 12 H 22 M 6.5 17.5 L 4.9 19.1 M 19.1 4.9 L 17.5 6.5"
    ),
    "calendar": svg_rect(3, 4.5, 18, 17, 2) + " M 8 2.5 V 6.5 M 16 2.5 V 6.5 M 3 10 H 21",
    "chevron-down": "M 6 9 L 12 15 L 18 9",
    "chevron-right": "M 9 18 L 15 12 L 9 6",
    "chevron-left": "M 15 18 L 9 12 L 15 6",
    "plus": "M 5 12 H 19 M 12 5 V 19",
    "x": "M 18 6 L 6 18 M 6 6 L 18 18",
    "check": "M 20 6.5 L 9.5 17 L 4.5 12",
    "check-circle-2": svg_circle(12, 12, 9.5) + " M 8.5 12 L 11 14.5 L 15.5 9.5",
    "alert-triangle": (
        "M 10.4 3.8 L 2.2 18.2 A 1.8 1.8 0 0 0 3.8 21 H 20.2 "
        "A 1.8 1.8 0 0 0 21.8 18.2 L 13.6 3.8 A 1.8 1.8 0 0 0 10.4 3.8 Z "
        "M 12 9.5 V 14 M 12 17.2 V 17.6"
    ),
    "alert-circle": svg_circle(12, 12, 9.5) + " M 12 8 V 13 M 12 16.5 V 17",
    "info": svg_circle(12, 12, 9.5) + " M 12 16.5 V 11.5 M 12 8 V 8.5",
    "download": "M 21 15 V 19 A 2 2 0 0 1 19 21 H 5 A 2 2 0 0 1 3 19 V 15 M 7 10 L 12 15 L 17 10 M 12 15 V 3",
    "printer": (
        "M 6 9 V 3 H 18 V 9 M 6 18 H 4 A 2 2 0 0 1 2 16 V 11 "
        "A 2 2 0 0 1 4 9 H 20 A 2 2 0 0 1 22 11 V 16 A 2 2 0 0 1 20 18 H 18 "
        + svg_rect(6, 14, 12, 7, 1)
    ),
    "rotate-ccw": "M 3.5 12 A 8.5 8.5 0 1 0 7 5.2 L 3.5 8.5 M 3.5 3.5 V 8.5 H 8.5",
    "scale": (
        "M 16 15 L 19 8 L 22 15 A 3.4 3.4 0 0 1 16 15 Z "
        "M 2 15 L 5 8 L 8 15 A 3.4 3.4 0 0 1 2 15 Z "
        "M 7 21 H 17 M 12 3 V 21 M 3.5 7 H 7 C 9 7 10.5 6 12 5 C 13.5 6 15 7 17 7 H 20.5"
    ),
    "trash-2": (
        "M 3 6 H 21 M 19 6 V 19.5 A 2 2 0 0 1 17 21.5 H 7 A 2 2 0 0 1 5 19.5 V 6 "
        "M 8.5 6 V 4 A 1.5 1.5 0 0 1 10 2.5 H 14 A 1.5 1.5 0 0 1 15.5 4 V 6 "
        "M 10 10.5 V 17 M 14 10.5 V 17"
    ),
    "refresh-cw": "M 3.5 12 A 8.5 8.5 0 0 1 18.2 6.2 L 20.5 8.5 M 20.5 3.5 V 8.5 H 15.5 M 20.5 12 A 8.5 8.5 0 0 1 5.8 17.8 L 3.5 15.5 M 8.5 15.5 H 3.5 V 20.5",
    "menu": "M 4 6 H 20 M 4 12 H 20 M 4 18 H 20",
    "loader-2": "M 20.5 12 A 8.5 8.5 0 1 1 12 3.5",
    "filter": "M 22 3.5 H 2 L 10 12.5 V 19 L 14 21 V 12.5 Z",
    "clock": svg_circle(12, 12, 9.5) + " M 12 7 V 12 L 15.5 14",
}

# ==============================================================================
# 3. FIGCRAFT DOCUMENT BUILDER ENGINE
# ==============================================================================


class DocBuilder:
    def __init__(self) -> None:
        self.cmds: list[dict[str, Any]] = []
        self._counter = 0
        self.vars: dict[str, str] = {}
        self.collections: dict[str, str] = {}
        self.text_styles: dict[str, str] = {}
        self.paint_styles: dict[str, str] = {}
        self.effect_styles: dict[str, str] = {}
        self.components: dict[str, str] = {}

    def _alias(self, prefix: str = "n") -> str:
        self._counter += 1
        return f"{prefix}_{self._counter}"

    def ref(self, alias: str) -> dict[str, str]:
        return {"$ref": alias}

    def add(self, command: str, params: dict[str, Any], prefix: str = "n") -> str:
        alias = self._alias(prefix)
        self.cmds.append({"command": command, "params": params, "as": alias})
        return alias

    def exec(self, command: str, params: dict[str, Any]) -> None:
        self.cmds.append({"command": command, "params": params})

    def page_create(self, name: str) -> str:
        return self.add("page.add", {"name": name}, "pg")

    def page_switch(self, page_alias: str) -> None:
        self.exec("page.select", {"id": self.ref(page_alias)})

    def page_rename_current(self, name: str) -> None:
        self.exec("page.rename", {"name": name})

    def frame(
        self,
        name: str,
        parent: str | None = None,
        x: float = 0,
        y: float = 0,
        width: float = 100,
        height: float = 100,
        props: dict[str, Any] | None = None,
        bind_var: tuple[str, str] | None = None,
        shadow: dict[str, Any] | None = None,
    ) -> str:
        params: dict[str, Any] = {
            "name": name,
            "x": x,
            "y": y,
            "width": width,
            "height": height,
            "select": False,
        }
        if parent is not None:
            params["parent"] = self.ref(parent)
        if props:
            params["props"] = props
        alias = self.add("frame.create", params, "fr")
        if bind_var:
            field, var_alias = bind_var
            self.bind_var(alias, field, var_alias)
        if shadow:
            self.add_drop_shadow(alias, shadow)
        return alias

    def rect(
        self,
        name: str,
        parent: str,
        x: float = 0,
        y: float = 0,
        width: float = 10,
        height: float = 10,
        props: dict[str, Any] | None = None,
    ) -> str:
        params: dict[str, Any] = {
            "type": "rectangle",
            "name": name,
            "parent": self.ref(parent),
            "x": x,
            "y": y,
            "width": width,
            "height": height,
            "select": False,
        }
        if props:
            params["props"] = props
        return self.add("node.create", params, "rc")

    def ellipse(
        self,
        name: str,
        parent: str,
        x: float = 0,
        y: float = 0,
        width: float = 8,
        height: float = 8,
        props: dict[str, Any] | None = None,
    ) -> str:
        params: dict[str, Any] = {
            "type": "ellipse",
            "name": name,
            "parent": self.ref(parent),
            "x": x,
            "y": y,
            "width": width,
            "height": height,
            "select": False,
        }
        if props:
            params["props"] = props
        return self.add("node.create", params, "el")

    def vector(
        self,
        name: str,
        parent: str,
        path: str,
        x: float = 0,
        y: float = 0,
        closed: bool = False,
        props: dict[str, Any] | None = None,
    ) -> str:
        params: dict[str, Any] = {
            "name": name,
            "parent": self.ref(parent),
            "path": path,
            "x": x,
            "y": y,
            "closed": closed,
            "select": False,
        }
        if props:
            params["props"] = props
        return self.add("vector.create", params, "vec")

    def text(
        self,
        content: str,
        parent: str,
        size: float = 13,
        color: str = "#0F172A",
        style: str = "Regular",
        family: str = "Inter",
        x: float = 0,
        y: float = 0,
        width: float | None = None,
        align: str = "left",
        letter_spacing: str | float | None = None,
        line_height: str | float | None = None,
        tnum: bool = False,
        zero: bool = False,
        truncate: bool = False,
        text_case: str | None = None,
        name: str = "Text",
        text_style: str | None = None,
        extra_props: dict[str, Any] | None = None,
    ) -> str:
        # Inter in FigCraft has: Regular, Italic, SemiBold, Bold (map Medium -> SemiBold)
        if family == "Inter" and style == "Medium":
            style = "SemiBold"
        props: dict[str, Any] = {
            "fontFamily": family,
            "fontStyle": style,
            "fontSize": size,
            "fill": color,
            "textAlign": align,
        }
        if width is not None:
            props["width"] = width
            props["textAutoResize"] = "truncate" if truncate else "height"
            if truncate:
                props["maxLines"] = 1
        if letter_spacing is not None:
            props["letterSpacing"] = letter_spacing
        if line_height is not None:
            props["lineHeight"] = line_height
        features: dict[str, bool] = {}
        if tnum:
            features["tnum"] = True
        if zero:
            features["zero"] = True
        if features:
            props["features"] = features
        if text_case:
            props["textCase"] = text_case
        if extra_props:
            props.update(extra_props)

        alias = self.add(
            "text.create",
            {
                "name": name,
                "parent": self.ref(parent),
                "text": content,
                "x": x,
                "y": y,
                "props": props,
                "select": False,
            },
            "tx",
        )
        if text_style and text_style in self.text_styles:
            self.exec(
                "style.apply",
                {"ids": [self.ref(alias)], "style": self.ref(self.text_styles[text_style])},
            )
            self.exec("node.set", {"ids": [self.ref(alias)], "fill": color, "textAlign": align})
        return alias

    def icon(
        self,
        icon_name: str,
        parent: str,
        size: float = 16,
        color: str = "#475569",
        stroke_weight: float | None = None,
        x: float = 0,
        y: float = 0,
        absolute: bool = False,
    ) -> str:
        raw_path = LUCIDE_24.get(icon_name, LUCIDE_24["info"])
        scaled_path = scale_svg_path(raw_path, size)
        sw = stroke_weight if stroke_weight is not None else (1.5 if size <= 16 else 1.75)
        frame_props: dict[str, Any] = {"fill": "none", "clip": False}
        if absolute:
            frame_props["absolute"] = True
        wrap = self.frame(f"ic/{icon_name}", parent, x, y, size, size, frame_props)
        self.vector(
            f"path/{icon_name}",
            wrap,
            scaled_path,
            x=0,
            y=0,
            closed=False,
            props={
                "fill": "none",
                "stroke": color,
                "strokeWeight": sw,
                "strokeCap": "round",
                "strokeJoin": "round",
            },
        )
        return wrap

    def brand_prism_mark(
        self,
        parent: str,
        size: float = 28,
        x: float = 0,
        y: float = 0,
        dark_surface: bool = False,
    ) -> str:
        """Crisp interlocking vector prism matching apps/web-spa/public/brand-logo.png."""
        bg = "#131F38" if dark_surface else "#F8FAFC"
        border = "#1E3A8A" if dark_surface else "#DBEAFE"
        box = self.frame(
            "BrandPrismMark",
            parent,
            x,
            y,
            size,
            size,
            {
                "fill": bg,
                "stroke": border,
                "strokeWeight": 1,
                "cornerRadius": max(6, round(size * 0.25)),
                "clip": True,
            },
        )
        s = size / 28.0
        # Back-right navy ribbon
        self.vector(
            "PrismNavy",
            box,
            f"M {14*s:g} {4.5*s:g} L {22.5*s:g} {10*s:g} L {18.5*s:g} {19*s:g} L {14*s:g} {16*s:g} Z",
            x=0,
            y=0,
            closed=True,
            props={"fill": "#1E3A8A", "stroke": "none"},
        )
        # Front-left cobalt facet
        self.vector(
            "PrismCobalt",
            box,
            f"M {14*s:g} {4.5*s:g} L {5.5*s:g} {10*s:g} L {9.5*s:g} {19*s:g} L {14*s:g} {13*s:g} Z",
            x=0,
            y=0,
            closed=True,
            props={"fill": "#2563EB", "stroke": "none"},
        )
        # Bottom emerald interlocking wedge
        self.vector(
            "PrismEmerald",
            box,
            f"M {9.5*s:g} {19*s:g} L {14*s:g} {14.5*s:g} L {18.5*s:g} {19*s:g} L {14*s:g} {23.5*s:g} Z",
            x=0,
            y=0,
            closed=True,
            props={"fill": "#10B981", "stroke": "none"},
        )
        # Crisp center highlight ridge
        self.vector(
            "PrismRidge",
            box,
            f"M {14*s:g} {6.2*s:g} L {17.5*s:g} {10.5*s:g} L {14*s:g} {13.5*s:g} L {10.5*s:g} {10.5*s:g} Z",
            x=0,
            y=0,
            closed=True,
            props={"fill": "#60A5FA", "stroke": "none"},
        )
        return box

    def place_image(
        self,
        path: Path,
        parent: str,
        x: float = 0,
        y: float = 0,
        width: float = 64,
        height: float = 64,
        mode: str = "fit",
        name: str = "Image",
    ) -> str:
        return self.add(
            "image.place",
            {
                "path": str(path),
                "parent": self.ref(parent),
                "x": x,
                "y": y,
                "width": width,
                "height": height,
                "mode": mode,
                "name": name,
            },
            "img",
        )

    def add_drop_shadow(self, target: str, shadow: dict[str, Any]) -> None:
        self.exec(
            "effect.add",
            {
                "ids": [self.ref(target)],
                "effect": {
                    "type": "dropShadow",
                    "color": shadow["color"],
                    "offset": shadow["offset"],
                    "radius": shadow["radius"],
                    "spread": shadow.get("spread", 0),
                },
            },
        )

    def bind_var(self, target: str, field: str, var_key: str) -> None:
        if var_key not in self.vars:
            return
        self.exec(
            "variable.bind",
            {
                "ids": [self.ref(target)],
                "field": field,
                "variable": self.ref(self.vars[var_key]),
            },
        )

    def set_mode(self, target: str, collection_key: str, mode: str) -> None:
        if collection_key not in self.collections:
            return
        self.exec(
            "variable.setMode",
            {
                "ids": [self.ref(target)],
                "collection": self.ref(self.collections[collection_key]),
                "mode": mode,
            },
        )

    def create_component(self, target: str, key: str | None = None) -> str:
        alias = self.add("component.create", {"ids": [self.ref(target)]}, "cmp")
        if key:
            self.components[key] = alias
        return alias

    def combine_variants(self, comp_aliases: list[str], key: str) -> str:
        alias = self.add(
            "component.combineVariants",
            {"ids": [self.ref(a) for a in comp_aliases]},
            "cset",
        )
        self.components[key] = alias
        return alias


# ==============================================================================
# 4. DESIGN SYSTEM REGISTRATION (Variables, Paint Styles, Text Styles, Effects)
# ==============================================================================


def register_tokens_and_styles(b: DocBuilder) -> None:
    col_sem = b.add(
        "variable.createCollection",
        {"name": "Semantic Theme", "modes": ["Light", "Dark"]},
        "col",
    )
    b.collections["semantic"] = col_sem

    sem_vars = {
        "bg/canvas": (LIGHT["canvas"], DARK["canvas"]),
        "bg/surface": (LIGHT["surface"], DARK["surface"]),
        "bg/surface-alt": (LIGHT["surface_alt"], DARK["surface_alt"]),
        "bg/surface-muted": (LIGHT["surface_muted"], DARK["surface_muted"]),
        "bg/navbar": (LIGHT["navbar"], DARK["navbar"]),
        "bg/sidebar": (LIGHT["sidebar"], DARK["sidebar"]),
        "border/default": (LIGHT["border"], DARK["border"]),
        "border/strong": (LIGHT["border_strong"], DARK["border_strong"]),
        "text/primary": (LIGHT["ink"], DARK["ink"]),
        "text/secondary": (LIGHT["ink_secondary"], DARK["ink_secondary"]),
        "text/muted": (LIGHT["ink_muted"], DARK["ink_muted"]),
        "brand/primary": (LIGHT["brand_primary"], DARK["brand_primary"]),
        "brand/soft": (LIGHT["brand_soft"], DARK["brand_soft"]),
        "brand/soft-border": (LIGHT["brand_soft_border"], DARK["brand_soft_border"]),
    }
    for vname, (lt, dk) in sem_vars.items():
        b.vars[vname] = b.add(
            "variable.create",
            {
                "collection": b.ref(col_sem),
                "name": vname,
                "type": "color",
                "values": {"Light": lt, "Dark": dk},
            },
            "var",
        )

    col_pal = b.add(
        "variable.createCollection",
        {"name": "Brand & Scale", "modes": ["Default"]},
        "col",
    )
    b.collections["palette"] = col_pal
    for step, hx in BRAND_SCALE.items():
        b.vars[f"brand/{step}"] = b.add(
            "variable.create",
            {
                "collection": b.ref(col_pal),
                "name": f"brand/{step}",
                "type": "color",
                "value": hx,
            },
            "var",
        )
    for step, hx in ACCENT_SCALE.items():
        b.vars[f"accent/{step}"] = b.add(
            "variable.create",
            {
                "collection": b.ref(col_pal),
                "name": f"accent/{step}",
                "type": "color",
                "value": hx,
            },
            "var",
        )

    # Text Styles
    text_specs = {
        "Display/H1": {
            "fontFamily": "Inter",
            "fontStyle": "Bold",
            "fontSize": 21,
            "letterSpacing": "-2%",
            "lineHeight": "125%",
        },
        "KPI/Hero": {
            "fontFamily": "Inter",
            "fontStyle": "Bold",
            "fontSize": 22,
            "letterSpacing": "-2%",
            "features": {"tnum": True},
        },
        "Heading/Card": {
            "fontFamily": "Inter",
            "fontStyle": "SemiBold",
            "fontSize": 14,
            "letterSpacing": "-1%",
        },
        "Body/Default": {
            "fontFamily": "Inter",
            "fontStyle": "Regular",
            "fontSize": 12.5,
            "lineHeight": "150%",
        },
        "Label/Upper": {
            "fontFamily": "Inter",
            "fontStyle": "SemiBold",
            "fontSize": 10.5,
            "letterSpacing": "4%",
        },
        "Mono/Code": {
            "fontFamily": "Menlo",
            "fontStyle": "Regular",
            "fontSize": 11.5,
            "features": {"tnum": True, "zero": True},
        },
        "Mono/Money": {
            "fontFamily": "Menlo",
            "fontStyle": "Bold",
            "fontSize": 12,
            "features": {"tnum": True, "zero": True},
        },
    }
    for sname, val in text_specs.items():
        b.text_styles[sname] = b.add(
            "style.create",
            {"type": "text", "name": sname, "value": val},
            "tsty",
        )

    # Paint Styles
    paint_specs = {
        "Brand/Primary 700": [{"type": "solid", "color": BRAND_SCALE["700"]}],
        "Brand/Deep 900": [{"type": "solid", "color": BRAND_SCALE["900"]}],
        "Accent/Orange 500": [{"type": "solid", "color": ACCENT_SCALE["500"]}],
        "Semantic/Success 600": [{"type": "solid", "color": "#059669"}],
        "Semantic/Warning 600": [{"type": "solid", "color": "#D97706"}],
        "Semantic/Danger 600": [{"type": "solid", "color": "#E11D48"}],
    }
    for pname, val in paint_specs.items():
        b.paint_styles[pname] = b.add(
            "style.create",
            {"type": "paint", "name": pname, "value": val},
            "psty",
        )

    # Effect Styles
    eff_specs = {
        "Elevation/Card": [
            {
                "type": "dropShadow",
                "color": "#0F172A0A",
                "offset": [0, 1],
                "radius": 3,
                "spread": 0,
            }
        ],
        "Elevation/Popover": [
            {
                "type": "dropShadow",
                "color": "#0F172A1C",
                "offset": [0, 12],
                "radius": 28,
                "spread": -6,
            }
        ],
        "Elevation/Modal": [
            {
                "type": "dropShadow",
                "color": "#0F172A38",
                "offset": [0, 22],
                "radius": 48,
                "spread": -10,
            }
        ],
    }
    for ename, val in eff_specs.items():
        b.effect_styles[ename] = b.add(
            "style.create",
            {"type": "effect", "name": ename, "value": val},
            "esty",
        )


# ==============================================================================
# 5. REUSABLE UI PRIMITIVES (Faithful to apps/web-spa/src/components/ui)
# ==============================================================================


def add_badge(
    b: DocBuilder,
    parent: str,
    label: str,
    tone: str,
    t: dict[str, Any],
    dot: bool = True,
    compact: bool = False,
    name: str | None = None,
) -> str:
    bg, bdr, dot_c, fg = t["badges"][tone]
    est_w = max(48, round(len(label) * (6.2 if compact else 6.6)) + (26 if dot else 18))
    h = 20 if compact else 22
    pill = b.frame(
        name or f"Badge/{label}",
        parent,
        0,
        0,
        est_w,
        h,
        {
            "fill": bg,
            "stroke": bdr,
            "strokeWeight": 1,
            "cornerRadius": 99,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
            "gap": 5,
            "padding": [0, 8 if compact else 9, 0, 8 if compact else 9],
            "sizingH": "hug",
        },
    )
    if dot:
        b.ellipse("Dot", pill, 0, 0, 5.5, 5.5, {"fill": dot_c, "stroke": "none"})
    b.text(
        label,
        pill,
        size=10.5 if compact else 11,
        color=fg,
        style="SemiBold",
        name="Label",
    )
    return pill


def add_button(
    b: DocBuilder,
    parent: str,
    label: str,
    t: dict[str, Any],
    variant: str = "primary",
    size: str = "md",
    icon_name: str | None = None,
    width: float | None = None,
    state: str = "default",
    name: str | None = None,
) -> str:
    dark = t["name"] == "Dark"
    heights = {"xs": 26, "sm": 32, "md": 36, "lg": 42}
    pads = {"xs": 9, "sm": 12, "md": 14, "lg": 18}
    fsizes = {"xs": 11, "sm": 12, "md": 12.5, "lg": 13.5}
    h = heights[size]
    px = pads[size]
    fsize = fsizes[size]

    if variant == "primary":
        bg = t["brand_primary_hover"] if state == "hover" else t["brand_primary"]
        bdr = "none"
        fg = "#FFFFFF"
    elif variant == "secondary":
        bg = t["surface_muted"] if state != "hover" else t["border"]
        bdr = t["border"]
        fg = t["ink"]
    elif variant == "outline":
        bg = t["surface"] if state != "hover" else t["surface_alt"]
        bdr = t["border_strong"] if state == "hover" else t["border"]
        fg = t["ink_secondary"]
    elif variant == "ghost":
        bg = t["surface_muted"] if state == "hover" else "none"
        bdr = "none"
        fg = t["ink_muted"] if state == "default" else t["ink"]
    elif variant == "danger":
        bg = "#BE123C" if state == "hover" else "#E11D48"
        bdr = "none"
        fg = "#FFFFFF"
    elif variant == "danger-soft":
        bg = "#4C051970" if dark else "#FFF1F2"
        bdr = "#9F1239" if dark else "#FECDD3"
        fg = "#FDA4AF" if dark else "#BE123C"
    elif variant == "success":
        bg = "#047857" if state == "hover" else "#059669"
        bdr = "none"
        fg = "#FFFFFF"
    else:
        bg, bdr, fg = t["surface"], t["border"], t["ink"]

    has_ic = bool(icon_name or state == "loading")
    est_w = width if width else max(60, round(len(label) * (fsize * 0.58)) + px * 2 + (22 if has_ic else 0))
    props: dict[str, Any] = {
        "fill": bg,
        "cornerRadius": 8 if size != "xs" else 6,
        "layout": "horizontal",
        "mainAlign": "center",
        "crossAlign": "center",
        "gap": 6,
        "padding": [0, px, 0, px],
    }
    if not width:
        props["sizingH"] = "hug"
    if bdr != "none":
        props["stroke"] = bdr
        props["strokeWeight"] = 1
    focus_shadow: dict[str, Any] | None = None
    btn_parent = parent
    if state == "disabled":
        props["opacity"] = 0.48
    elif state == "focus":
        is_solid = variant in ("primary", "danger", "success")
        props["stroke"] = "#93C5FD" if is_solid else ("#60A5FA" if dark else "#3B82F6")
        props["strokeWeight"] = 3 if is_solid else 2
        props["strokeAlign"] = "outside"
        if bg == "none":
            # Transparent ghost buttons bleed dropShadow across their interior in FigCraft;
            # use an unclipped outer halo stroke wrapper so the button interior stays clear.
            halo_props: dict[str, Any] = {
                "fill": "none",
                "cornerRadius": props["cornerRadius"],
                "layout": "horizontal",
                "mainAlign": "center",
                "crossAlign": "center",
                "sizingV": "hug",
                "stroke": "#60A5FA44" if dark else "#3B82F638",
                "strokeWeight": 5,
                "strokeAlign": "outside",
                "clip": False,
            }
            if not width:
                halo_props["sizingH"] = "hug"
            btn_parent = b.frame(f"FocusHalo/{variant}/{label}", parent, 0, 0, est_w, h, halo_props)
        else:
            focus_shadow = {"color": "#3B82F666", "offset": [0, 0], "radius": 6, "spread": 0}

    btn = b.frame(name or f"Button/{variant}/{label}", btn_parent, 0, 0, est_w, h, props, shadow=focus_shadow)
    if state == "loading":
        b.icon("loader-2", btn, size=14, color=fg, stroke_weight=1.9)
    elif icon_name:
        b.icon(icon_name, btn, size=13.5 if size in ("xs", "sm") else 14.5, color=fg)
    b.text(label, btn, size=fsize, color=fg, style="SemiBold", name="Label")
    return btn


def add_input_field(
    b: DocBuilder,
    parent: str,
    label: str | None,
    value: str,
    t: dict[str, Any],
    width: float = 260,
    input_h: float = 36,
    left_icon: str | None = None,
    right_icon: str | None = None,
    currency_chip: str | None = None,
    state: str = "default",
    helper: str | None = None,
    mono: bool = False,
    right_align: bool = False,
    required: bool = False,
) -> str:
    dark = t["name"] == "Dark"
    box_h = input_h + (20 if label else 0) + (18 if helper else 0)
    wrap = b.frame(
        f"Field/{label or value}",
        parent,
        0,
        0,
        width,
        box_h,
        {"fill": "none", "layout": "vertical", "gap": 5},
    )
    if label:
        lbl_row = b.frame(
            "LabelRow",
            wrap,
            0,
            0,
            width,
            15,
            {"fill": "none", "layout": "horizontal", "gap": 3, "crossAlign": "center"},
        )
        b.text(label, lbl_row, size=11.5, color=t["ink_secondary"], style="SemiBold")
        if required:
            b.text("*", lbl_row, size=11.5, color="#E11D48", style="SemiBold")

    bg = t["surface_muted"] if state == "disabled" else (t["surface_alt"] if state == "readonly" else t["surface"])
    if state == "focus":
        bdr = "#2563EB" if not dark else "#3B82F6"
        sw = 1.5
    elif state == "error":
        bdr = "#E11D48" if not dark else "#FB7185"
        sw = 1.5
    else:
        bdr = t["border"]
        sw = 1

    inp = b.frame(
        "InputBox",
        wrap,
        0,
        0,
        width,
        input_h,
        {
            "fill": bg,
            "stroke": bdr,
            "strokeWeight": sw,
            "cornerRadius": 8,
            "layout": "horizontal",
            "crossAlign": "center",
            "mainAlign": "spaceBetween",
            "gap": 8,
            "padding": [0, 10, 0, 0 if currency_chip else 10],
            "clip": True,
        },
    )
    if state == "disabled":
        b.exec("node.set", {"ids": [b.ref(inp)], "opacity": 0.65})

    if currency_chip:
        chip = b.frame(
            "CurrencyChip",
            inp,
            0,
            0,
            40,
            input_h - 2,
            {
                "fill": t["surface_muted"],
                "strokeRightWeight": 1,
                "stroke": t["border"],
                "layout": "horizontal",
                "mainAlign": "center",
                "crossAlign": "center",
            },
        )
        b.text(currency_chip, chip, size=10.5, color=t["ink_secondary"], style="Bold", family="Menlo", zero=True)

    left_grp = b.frame(
        "Left",
        inp,
        0,
        0,
        width - (64 if currency_chip else 28),
        input_h,
        {
            "fill": "none",
            "layout": "horizontal",
            "crossAlign": "center",
            "gap": 6,
            "sizingH": "fill",
            "mainAlign": "max" if right_align else "min",
        },
    )
    if left_icon:
        b.icon(left_icon, left_grp, size=14, color=t["ink_muted"])
    txt_color = t["ink_muted"] if state in ("placeholder", "disabled") else t["ink"]
    b.text(
        value,
        left_grp,
        size=12,
        color=txt_color,
        style="Bold" if (right_align and mono) else "Regular",
        family="Menlo" if mono else "Inter",
        tnum=mono,
        zero=mono,
        align="right" if right_align else "left",
        truncate=True,
    )
    if right_icon:
        b.icon(right_icon, inp, size=14, color=t["ink_muted"])

    if helper:
        h_color = ("#FB7185" if dark else "#E11D48") if state == "error" else t["ink_muted"]
        b.text(helper, wrap, size=10.5, color=h_color, style="SemiBold" if state == "error" else "Regular")
    return wrap


# ==============================================================================
# 6. SHELL: NAVBAR + 16-MODULE SIDEBAR (Navbar.tsx + Sidebar.tsx + permissions.ts)
# ==============================================================================


def add_navbar(b: DocBuilder, parent: str, t: dict[str, Any], portal: str = "Backoffice") -> str:
    dark = t["name"] == "Dark"
    nav = b.frame(
        "Navbar",
        parent,
        0,
        0,
        1440,
        56,
        {
            "fill": t["navbar"],
            "stroke": t["border"],
            "strokeBottomWeight": 1,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 20, 0, 16],
        },
        bind_var=("fill", "bg/navbar"),
    )

    # Left: Prism Brand + PortalSwitcher + CmdK Trigger
    left = b.frame(
        "NavLeft",
        nav,
        0,
        0,
        820,
        40,
        {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 14},
    )
    brand_wrap = b.frame(
        "BrandLockup",
        left,
        0,
        0,
        220,
        36,
        {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10},
    )
    b.brand_prism_mark(brand_wrap, size=28, dark_surface=dark)
    brand_txt = b.frame(
        "BrandText",
        brand_wrap,
        0,
        0,
        178,
        30,
        {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 7},
    )
    b.text("Enterprise ERP", brand_txt, size=13.5, color=t["ink"], style="Bold", letter_spacing="-1%")
    ver = b.frame(
        "VerBadge",
        brand_txt,
        0,
        0,
        34,
        18,
        {
            "fill": t["brand_soft"],
            "stroke": t["brand_soft_border"],
            "strokeWeight": 1,
            "cornerRadius": 5,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    b.text("v2.4", ver, size=9.5, color=t["brand_soft_ink"], style="Bold", family="Menlo")

    # Hairline divider
    b.rect("Div1", left, 0, 0, 1, 20, {"fill": t["border"]})

    # PortalSwitcher (Backoffice / Admin / BI Report with real icons)
    ps = b.frame(
        "PortalSwitcher",
        left,
        0,
        0,
        282,
        32,
        {
            "fill": t["surface_muted"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 8,
            "layout": "horizontal",
            "crossAlign": "center",
            "gap": 2,
            "padding": 3,
        },
    )
    portals = [
        ("Backoffice", "briefcase"),
        ("Admin", "shield-check"),
        ("BI Report", "bar-chart-3"),
    ]
    for pname, pico in portals:
        active = pname == portal
        tab = b.frame(
            f"Portal/{pname}",
            ps,
            0,
            0,
            92 if pname == "Backoffice" else (78 if pname == "Admin" else 94),
            26,
            {
                "fill": t["surface"] if active else "none",
                "cornerRadius": 6,
                "layout": "horizontal",
                "mainAlign": "center",
                "crossAlign": "center",
                "gap": 5,
                "padding": [0, 8, 0, 8],
            },
            shadow=t["shadow_card"] if active else None,
        )
        fg = t["ink"] if active else t["ink_muted"]
        b.icon(pico, tab, size=12.5, color=t["brand_primary"] if active else fg)
        b.text(pname, tab, size=11.5, color=fg, style="SemiBold" if active else "Regular")

    # CommandPaletteTrigger (Cmd+K)
    cmdk = b.frame(
        "CommandPaletteTrigger",
        left,
        0,
        0,
        228,
        32,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 8,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 8, 0, 10],
        },
    )
    cmd_l = b.frame("L", cmdk, 0, 0, 166, 20, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 7})
    b.icon("search", cmd_l, size=13, color=t["ink_muted"])
    b.text("Cari modul, jurnal, akun...", cmd_l, size=11.5, color=t["ink_muted"])
    kbd = b.frame(
        "Kbd",
        cmdk,
        0,
        0,
        28,
        20,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 5,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    b.text("⌘K", kbd, size=10, color=t["ink_muted"], style="Bold", family="Menlo")

    # Right cluster: OrgSwitcher + SSE Live + Locale + Theme + Notification Bell + Profile
    right = b.frame(
        "NavRight",
        nav,
        0,
        0,
        590,
        36,
        {
            "fill": "none",
            "layout": "horizontal",
            "mainAlign": "max",
            "crossAlign": "center",
            "gap": 8,
        },
    )

    org = b.frame(
        "OrgSwitcher",
        right,
        0,
        0,
        216,
        32,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 8,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "gap": 6,
            "padding": [0, 9, 0, 9],
        },
    )
    org_l = b.frame("L", org, 0, 0, 180, 20, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 6})
    b.icon("building-2", org_l, size=13.5, color=t["brand_primary"])
    b.text("PT-SN · Sinergi Nusantara", org_l, size=11.5, color=t["ink"], style="SemiBold")
    b.icon("chevron-down", org, size=12, color=t["ink_muted"])

    # LiveSSEBadge
    add_badge(b, right, "SSE Live", "success", t, dot=True, compact=True, name="LiveSSEBadge")

    # Locale pill
    loc = b.frame(
        "LocaleToggle",
        right,
        0,
        0,
        32,
        30,
        {
            "fill": "none",
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 7,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    b.text("ID", loc, size=11, color=t["ink_secondary"], style="SemiBold", family="Menlo")

    # Ghost Theme Toggle (Crisp Moon in Light, Crisp Sun with rays in Dark)
    theme_btn = b.frame(
        "ThemeToggle",
        right,
        0,
        0,
        32,
        32,
        {
            "fill": "none",
            "cornerRadius": 8,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    if dark:
        b.icon("sun", theme_btn, size=15.5, color="#FBBF24", stroke_weight=1.7)
    else:
        b.icon("moon", theme_btn, size=15.5, color=t["ink_muted"], stroke_weight=1.6)

    # NotificationPopover Bell + Badge 3
    bell_wrap = b.frame(
        "NotificationTrigger",
        right,
        0,
        0,
        34,
        32,
        {"fill": "none", "cornerRadius": 8, "clip": False},
    )
    b.icon("bell", bell_wrap, size=16, color=t["ink_secondary"], x=9, y=8, stroke_weight=1.6)
    notif_badge = b.frame(
        "UnreadDot",
        bell_wrap,
        18,
        3,
        14,
        14,
        {
            "fill": "#E11D48",
            "stroke": t["navbar"],
            "strokeWeight": 1.5,
            "cornerRadius": 99,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    b.text("3", notif_badge, size=8.5, color="#FFFFFF", style="Bold")

    # Hairline divider
    b.rect("Div2", right, 0, 0, 1, 20, {"fill": t["border"]})

    # UserProfileMenu (AS badge + full name + ChevronDown)
    user_btn = b.frame(
        "UserProfileMenu",
        right,
        0,
        0,
        134,
        32,
        {
            "fill": "none",
            "cornerRadius": 8,
            "layout": "horizontal",
            "crossAlign": "center",
            "gap": 7,
            "padding": [0, 4, 0, 2],
        },
    )
    av = b.frame(
        "Avatar",
        user_btn,
        0,
        0,
        26,
        26,
        {
            "fill": t["brand_primary"],
            "cornerRadius": 6,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    b.text("AS", av, size=10.5, color="#FFFFFF", style="Bold")
    b.text("Agga Sugitha", user_btn, size=11.5, color=t["ink"], style="SemiBold")
    b.icon("chevron-down", user_btn, size=12, color=t["ink_muted"])

    return nav


SIDEBAR_GROUPS = [
    (
        "RINGKASAN",
        [
            ("Dashboard Eksekutif", "layout-dashboard", None),
        ],
    ),
    (
        "KEUANGAN & AKUNTANSI",
        [
            ("Buku Besar & Jurnal", "book-open", None),
            ("Bagan Akun & Periode", "layers", None),
            ("Hutang Usaha (AP)", "arrow-down-left", "6"),
            ("Piutang Usaha (AR)", "arrow-up-right", None),
            ("Pajak & e-Faktur", "receipt", None),
            ("Kas, Bank & Valas", "landmark", None),
            ("Mata Uang & Kurs FX", "coins", None),
        ],
    ),
    (
        "RANTAI PASOK & OPERASIONAL",
        [
            ("Pengadaan (PR/PO)", "shopping-cart", None),
            ("Persediaan & Gudang", "package", None),
            ("Penjualan (SO)", "trending-up", None),
            ("Manufaktur & BOM", "factory", None),
        ],
    ),
    (
        "SDM & PAYROLL",
        [
            ("Sumber Daya Manusia", "users", None),
            ("Penggajian (Payroll)", "wallet", None),
        ],
    ),
    (
        "TATA KELOLA & MASTER",
        [
            ("Otorisasi & Persetujuan", "shield-check", "3"),
            ("Master Partner & UoM", "database", None),
        ],
    ),
]


def add_sidebar(
    b: DocBuilder,
    parent: str,
    t: dict[str, Any],
    active_label: str = "Dashboard Eksekutif",
) -> str:
    dark = t["name"] == "Dark"
    sb = b.frame(
        "Sidebar",
        parent,
        0,
        56,
        256,
        884,
        {
            "fill": t["sidebar"],
            "stroke": t["border"],
            "strokeRightWeight": 1,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "padding": [12, 10, 12, 10],
            "clip": True,
        },
        bind_var=("fill", "bg/sidebar"),
    )

    nav_list = b.frame(
        "NavGroups",
        sb,
        0,
        0,
        236,
        742,
        {"fill": "none", "layout": "vertical", "gap": 9},
    )

    for grp_title, items in SIDEBAR_GROUPS:
        grp = b.frame(
            f"Group/{grp_title}",
            nav_list,
            0,
            0,
            236,
            18 + len(items) * 33,
            {"fill": "none", "layout": "vertical", "gap": 2},
        )
        hdr = b.frame(
            "GroupHeader",
            grp,
            0,
            0,
            236,
            16,
            {"fill": "none", "layout": "horizontal", "crossAlign": "center", "padding": [0, 8, 0, 8]},
        )
        b.text(
            grp_title,
            hdr,
            size=9.5,
            color=t["ink_subtle"],
            style="Bold",
            letter_spacing="5%",
        )

        for label, ico, count in items:
            active = label == active_label
            row = b.frame(
                f"Nav/{label}",
                grp,
                0,
                0,
                236,
                31,
                {
                    "fill": t["brand_soft"] if active else "none",
                    "stroke": t["brand_soft_border"] if active else "none",
                    "strokeWeight": 1 if active else 0,
                    "cornerRadius": 7,
                    "layout": "horizontal",
                    "mainAlign": "spaceBetween",
                    "crossAlign": "center",
                    "padding": [0, 8, 0, 6],
                },
            )
            left = b.frame(
                "L",
                row,
                0,
                0,
                194,
                22,
                {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8},
            )
            if active:
                b.rect(
                    "ActiveBar",
                    left,
                    0,
                    0,
                    3,
                    16,
                    {"fill": t["brand_primary"] if not dark else "#60A5FA", "cornerRadius": 2},
                )
            else:
                b.rect("SpacerBar", left, 0, 0, 3, 16, {"fill": "none"})
            ico_c = (t["brand_primary"] if not dark else "#60A5FA") if active else t["ink_muted"]
            lbl_c = t["brand_soft_ink"] if active else t["ink_secondary"]
            b.icon(ico, left, size=14.5, color=ico_c, stroke_weight=1.75 if active else 1.5)
            b.text(
                label,
                left,
                size=12,
                color=lbl_c,
                style="SemiBold" if active else "Regular",
                truncate=True,
                width=162,
            )
            if count:
                pill = b.frame(
                    "CountPill",
                    row,
                    0,
                    0,
                    20,
                    17,
                    {
                        "fill": "#1E3A8A" if (active and dark) else (t["brand_soft"] if active else t["surface_muted"]),
                        "cornerRadius": 99,
                        "layout": "horizontal",
                        "mainAlign": "center",
                        "crossAlign": "center",
                    },
                )
                b.text(
                    count,
                    pill,
                    size=9.5,
                    color=t["brand_soft_ink"] if active else t["ink_muted"],
                    style="Bold",
                    family="Menlo",
                )

    # Sidebar Footer Status Card
    foot = b.frame(
        "SidebarFooter",
        sb,
        0,
        0,
        236,
        70,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 9,
            "layout": "vertical",
            "gap": 4,
            "padding": [10, 12, 10, 12],
        },
    )
    top_f = b.frame("FTop", foot, 0, 0, 212, 16, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 6})
    b.ellipse("OnlineDot", top_f, 0, 0, 6, 6, {"fill": "#10B981"})
    b.text("Tri-Currency ADR-014", top_f, size=11, color=t["ink"], style="SemiBold")
    b.text("Fungsional: IDR · Konsol: USD", foot, size=10.5, color=t["ink_muted"], family="Menlo", zero=True)
    b.text("Periode 2026-11 · Terbuka", foot, size=10.5, color=t["ink_muted"], style="SemiBold")
    return sb


# ==============================================================================
# 7. KPI CARDS & MATHEMATICALLY HONEST DATA VISUALIZATIONS
# ==============================================================================


def add_kpi_card(
    b: DocBuilder,
    parent: str,
    title: str,
    value: str,
    sub_text: str,
    delta_label: str,
    delta_tone: str,
    accent: str,
    icon_name: str,
    t: dict[str, Any],
    width: float = 272,
    height: float = 112,
) -> str:
    ic_bg, ic_bdr, ic_fg = t["kpi_icon"][accent]
    card = b.frame(
        f"KpiCard/{title}",
        parent,
        0,
        0,
        width,
        height,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 12,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "padding": [14, 16, 13, 16],
        },
        bind_var=("fill", "bg/surface"),
        shadow=t["shadow_card"],
    )
    top = b.frame(
        "TopRow",
        card,
        0,
        0,
        width - 32,
        28,
        {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"},
    )
    b.text(
        title,
        top,
        size=10.5,
        color=t["ink_muted"],
        style="Bold",
        letter_spacing="4%",
        text_style="Label/Upper",
    )
    ic_box = b.frame(
        "IconWrap",
        top,
        0,
        0,
        28,
        28,
        {
            "fill": ic_bg,
            "stroke": ic_bdr,
            "strokeWeight": 1,
            "cornerRadius": 8,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    b.icon(icon_name, ic_box, size=14.5, color=ic_fg, stroke_weight=1.7)

    # Hero Value: crisp Inter Bold + tabular numerals (tnum), NEVER clunky Menlo!
    b.text(
        value,
        card,
        size=22,
        color=t["ink"],
        style="Bold",
        letter_spacing="-2%",
        tnum=True,
        name="HeroValue",
        text_style="KPI/Hero",
    )

    bot = b.frame(
        "BottomRow",
        card,
        0,
        0,
        width - 32,
        20,
        {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "gap": 6},
    )
    add_badge(b, bot, delta_label, delta_tone, t, dot=False, compact=True, name="Delta")
    b.text(sub_text, bot, size=10.5, color=t["ink_muted"], align="right")
    return card


def add_gl_bar_chart(b: DocBuilder, parent: str, t: dict[str, Any], width: float = 560, height: float = 286) -> str:
    dark = t["name"] == "Dark"
    palette = CHART_COLORS_DARK if dark else CHART_COLORS
    card = b.frame(
        "Chart/GLByCategory",
        parent,
        0,
        0,
        width,
        height,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 12,
            "clip": True,
        },
        bind_var=("fill", "bg/surface"),
        shadow=t["shadow_card"],
    )

    # Header
    hdr = b.frame(
        "ChartHeader",
        card,
        18,
        16,
        width - 36,
        38,
        {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "min"},
    )
    hl = b.frame("HL", hdr, 0, 0, 380, 36, {"fill": "none", "layout": "vertical", "gap": 2})
    b.text("Saldo Buku Besar per Kategori Akun", hl, size=14, color=t["ink"], style="SemiBold")
    b.text("Neraca & Laba Rugi konsolidasi · A = L + E (Miliar IDR)", hl, size=11, color=t["ink_muted"])
    add_badge(b, hdr, "A = L + E · Valid", "brand", t, dot=True, compact=True)

    # Exact Linear Scale Plot Geometry (0..60M -> 144px vertical plot!)
    plot_left = 68
    plot_right = width - 20
    plot_w = plot_right - plot_left
    plot_top = 86
    plot_h = 144
    plot_bot = plot_top + plot_h
    max_val = 60.0

    # Gridlines at 60M, 45M, 30M, 15M, 0
    for val, lbl in [(60.0, "Rp 60M"), (45.0, "Rp 45M"), (30.0, "Rp 30M"), (15.0, "Rp 15M"), (0.0, "Rp 0")]:
        gy = plot_bot - (val / max_val) * plot_h
        b.text(
            lbl,
            card,
            size=10,
            color=t["ink_subtle"],
            family="Menlo",
            tnum=True,
            x=18,
            y=gy - 6,
            width=44,
            align="right",
        )
        b.rect(
            f"Grid/{val:g}",
            card,
            plot_left,
            gy,
            plot_w,
            1,
            {"fill": t["border_strong"] if val == 0 else t["gridline"]},
        )

    # Balanced Accounting Data (Asset = 48.25, Liability = 14.82, Equity = 33.43, Revenue = 29.40, Expense = 19.15)
    bars = [
        ("Aset", 48.25, palette["Asset"]),
        ("Kewajiban", 14.82, palette["Liability"]),
        ("Ekuitas", 33.43, palette["Equity"]),
        ("Pendapatan", 29.40, palette["Revenue"]),
        ("Beban", 19.15, palette["Expense"]),
    ]
    slot_w = plot_w / len(bars)
    bar_w = 52
    for idx, (cat, val, col) in enumerate(bars):
        cx = plot_left + idx * slot_w + slot_w / 2.0
        bh = round((val / max_val) * plot_h, 1)
        bx = cx - bar_w / 2.0
        by = plot_bot - bh

        # Soft gradient fill so bars look crafted, not flat solid blocks
        b.rect(
            f"Bar/{cat}",
            card,
            bx,
            by,
            bar_w,
            bh,
            {
                "fills": [
                    {
                        "type": "linearGradient",
                        "stops": [col, f"{col}B8"],
                        "from": [0.5, 0],
                        "to": [0.5, 1],
                    }
                ],
                "cornerRadius": [6, 6, 0, 0],
            },
        )
        # Centered value label strictly above bar top
        b.text(
            f"{val:.2f}".replace(".", ",") + " M",
            card,
            size=10.5,
            color=t["ink"],
            style="Bold",
            family="Menlo",
            tnum=True,
            x=cx - 38,
            y=by - 17,
            width=76,
            align="center",
        )
        # Centered X-axis category label
        b.text(
            cat,
            card,
            size=11,
            color=t["ink_secondary"],
            style="SemiBold",
            x=cx - 44,
            y=plot_bot + 10,
            width=88,
            align="center",
        )
    return card


def _catmull_rom_to_bezier_path(pts: list[tuple[float, float]]) -> str:
    """Convert ordered (x, y) coordinates into a smooth C1 cubic Bezier SVG path."""
    if len(pts) < 2:
        return ""
    segs = [f"M {pts[0][0]:.2f} {pts[0][1]:.2f}"]
    n = len(pts)
    for i in range(n - 1):
        p0 = pts[i - 1] if i > 0 else pts[0]
        p1 = pts[i]
        p2 = pts[i + 1]
        p3 = pts[i + 2] if i + 2 < n else pts[n - 1]
        c1x = p1[0] + (p2[0] - p0[0]) / 6.0
        c1y = p1[1] + (p2[1] - p0[1]) / 6.0
        c2x = p2[0] - (p3[0] - p1[0]) / 6.0
        c2y = p2[1] - (p3[1] - p1[1]) / 6.0
        segs.append(f"C {c1x:.2f} {c1y:.2f}, {c2x:.2f} {c2y:.2f}, {p2[0]:.2f} {p2[1]:.2f}")
    return " ".join(segs)


def add_sales_area_chart(b: DocBuilder, parent: str, t: dict[str, Any], width: float = 560, height: float = 286) -> str:
    dark = t["name"] == "Dark"
    teal = "#2DD4BF" if dark else "#0F766E"
    teal_grad_top = "#2DD4BF4A" if dark else "#0F766E36"
    teal_grad_bot = "#2DD4BF00" if dark else "#0F766E00"

    card = b.frame(
        "Chart/SalesTrend",
        parent,
        0,
        0,
        width,
        height,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 12,
            "clip": True,
        },
        bind_var=("fill", "bg/surface"),
        shadow=t["shadow_card"],
    )

    # Header + Segmented Toggle [Penjualan | Persediaan] + [Tabel]
    hdr = b.frame(
        "ChartHeader",
        card,
        18,
        16,
        width - 36,
        38,
        {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "min"},
    )
    hl = b.frame("HL", hdr, 0, 0, 320, 36, {"fill": "none", "layout": "vertical", "gap": 2})
    b.text("Tren Penjualan per Periode", hl, size=14, color=t["ink"], style="SemiBold")
    b.text("Pendapatan penjualan 7 bulan terakhir · Mei – Nov 2026", hl, size=11, color=t["ink_muted"])

    ctrl = b.frame("Ctrl", hdr, 0, 0, 214, 28, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 6})
    seg = b.frame(
        "MetricToggle",
        ctrl,
        0,
        0,
        148,
        26,
        {
            "fill": t["surface_muted"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 7,
            "layout": "horizontal",
            "crossAlign": "center",
            "padding": 2,
            "gap": 2,
        },
    )
    p1 = b.frame(
        "Tab/Penjualan",
        seg,
        0,
        0,
        70,
        22,
        {"fill": t["surface"], "cornerRadius": 5, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"},
        shadow=t["shadow_card"],
    )
    b.text("Penjualan", p1, size=10.5, color=t["ink"], style="SemiBold")
    p2 = b.frame(
        "Tab/Persediaan",
        seg,
        0,
        0,
        72,
        22,
        {"fill": "none", "cornerRadius": 5, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"},
    )
    b.text("Persediaan", p2, size=10.5, color=t["ink_muted"])

    tbl_btn = b.frame(
        "Btn/Tabel",
        ctrl,
        0,
        0,
        56,
        26,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 7,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
        },
    )
    b.text("Tabel", tbl_btn, size=10.5, color=t["ink_secondary"], style="SemiBold")

    # Identical Y baseline & plot height as GL Bar Chart so horizontal rhythm aligns!
    plot_left = 68
    plot_right = width - 24
    plot_w = plot_right - plot_left
    plot_top = 86
    plot_h = 144
    plot_bot = plot_top + plot_h
    max_val = 9.0

    for val, lbl in [(9.0, "Rp 9M"), (6.0, "Rp 6M"), (3.0, "Rp 3M"), (0.0, "Rp 0")]:
        gy = plot_bot - (val / max_val) * plot_h
        b.text(
            lbl,
            card,
            size=10,
            color=t["ink_subtle"],
            family="Menlo",
            tnum=True,
            x=18,
            y=gy - 6,
            width=44,
            align="right",
        )
        b.rect(
            f"Grid/{val:g}",
            card,
            plot_left,
            gy,
            plot_w,
            1,
            {"fill": t["border_strong"] if val == 0 else t["gridline"]},
        )

    series = [
        ("Mei", 3.85),
        ("Jun", 4.42),
        ("Jul", 4.18),
        ("Agu", 5.25),
        ("Sep", 5.42),
        ("Okt", 6.42),
        ("Nov", 6.95),
    ]
    pts: list[tuple[float, float]] = []
    # Coordinates relative to (plot_left, plot_top) so gradient bounding box fits curve neatly
    rel_pts: list[tuple[float, float]] = []
    pad_x = 18
    usable_w = plot_w - pad_x * 2
    for idx, (mon, val) in enumerate(series):
        px = pad_x + (idx / (len(series) - 1)) * usable_w
        py = plot_h - (val / max_val) * plot_h
        rel_pts.append((px, py))
        pts.append((plot_left + px, plot_top + py))
        b.text(
            mon,
            card,
            size=11,
            color=t["ink"] if mon == "Okt" else t["ink_muted"],
            style="Bold" if mon == "Okt" else "SemiBold",
            x=plot_left + px - 18,
            y=plot_bot + 10,
            width=36,
            align="center",
        )

    stroke_d = _catmull_rom_to_bezier_path(rel_pts)
    area_d = f"{stroke_d} L {rel_pts[-1][0]:.2f} {plot_h:.2f} L {rel_pts[0][0]:.2f} {plot_h:.2f} Z"

    # Filled Area Gradient
    b.vector(
        "SalesAreaFill",
        card,
        area_d,
        x=plot_left,
        y=plot_top,
        closed=True,
        props={
            "fills": [
                {
                    "type": "linearGradient",
                    "stops": [teal_grad_top, teal_grad_bot],
                    "from": [0.5, 0],
                    "to": [0.5, 1],
                }
            ]
        },
    )
    # Smooth Curve Stroke
    b.vector(
        "SalesLine",
        card,
        stroke_d,
        x=plot_left,
        y=plot_top,
        closed=False,
        props={"fill": "none", "stroke": teal, "strokeWeight": 2.4, "strokeCap": "round", "strokeJoin": "round"},
    )

    # Dashed Crosshair & Halo Dot on Okt (index 5)
    okt_x, okt_y = pts[5]
    b.vector(
        "CrosshairOkt",
        card,
        f"M 0 0 V {plot_bot - okt_y:.2f}",
        x=okt_x,
        y=okt_y,
        closed=False,
        props={"fill": "none", "stroke": teal, "strokeWeight": 1.2, "dashes": [3, 3]},
    )

    # Subtle dots on all vertices + Prominent halo on Okt
    for idx, (vx, vy) in enumerate(pts):
        if idx == 5:
            b.ellipse(
                "HaloOkt",
                card,
                vx - 9,
                vy - 9,
                18,
                18,
                {"fill": "#2DD4BF30" if dark else "#0F766E24"},
            )
            b.ellipse(
                "DotOkt",
                card,
                vx - 4.5,
                vy - 4.5,
                9,
                9,
                {"fill": t["surface"], "stroke": teal, "strokeWeight": 2.2},
            )
        else:
            b.ellipse(
                f"Dot/{idx}",
                card,
                vx - 2.8,
                vy - 2.8,
                5.6,
                5.6,
                {"fill": t["surface"], "stroke": teal, "strokeWeight": 1.5},
            )

    # Floating Tooltip positioned cleanly above curve & left of Okt (zero overlap with curve!)
    tip = b.frame(
        "TooltipOkt",
        card,
        okt_x - 168,
        plot_top - 6,
        148,
        46,
        {
            "fill": t["tooltip_bg"],
            "stroke": t["tooltip_border"],
            "strokeWeight": 1,
            "cornerRadius": 9,
            "layout": "vertical",
            "gap": 2,
            "padding": [6, 10, 6, 10],
        },
        shadow=t["shadow_popover"],
    )
    tr1 = b.frame("TTop", tip, 0, 0, 128, 14, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Okt 2026", tr1, size=10.5, color=t["tooltip_muted"], style="SemiBold")
    b.text("+18,4% MoM", tr1, size=10, color="#34D399", style="Bold", family="Menlo")
    b.text("Rp 6.420.000.000", tip, size=11.5, color=t["tooltip_ink"], style="Bold", tnum=True)

    return card


# ==============================================================================
# 8. SHARED JOURNAL TABLE & WORKSPACE BASE (Used across Screens 02, 03, 04)
# ==============================================================================

JOURNAL_ROWS = [
    ("JRN-2026-0842", "15 Nov 2026", "Pelunasan inv cloud & lisensi Q4 · INV-VND-1192", "USD @ 15.875", "190.500.000", "190.500.000", "Posted", "success"),
    ("JRN-2026-0841", "15 Nov 2026", "Penerimaan termin piutang proyek PT Wijaya Karya", "IDR @ 1,00", "425.000.000", "425.000.000", "Posted", "success"),
    ("JRN-2026-0840", "14 Nov 2026", "Pembelian bahan baku resin impor · PO-2026-0318", "SGD @ 11.840", "118.400.000", "118.400.000", "Posted", "success"),
    ("JRN-2026-0839", "14 Nov 2026", "Revaluasi kurs tengah akhir pekan kas valas", "USD @ 15.875", "84.320.000", "84.320.000", "Menunggu", "warning"),
    ("JRN-2026-0838", "13 Nov 2026", "Akrual biaya logistik & bea masuk Priok", "IDR @ 1,00", "67.850.000", "67.850.000", "Posted", "success"),
    ("JRN-2026-0837", "12 Nov 2026", "Jurnal pembalik selisih stok Gudang Cikarang B", "IDR @ 1,00", "41.200.000", "41.200.000", "Dibalik", "danger"),
    ("JRN-2026-0836", "11 Nov 2026", "Pembayaran PPN Masa & PPh 23 · NTPN-8849201", "IDR @ 1,00", "154.900.000", "154.900.000", "Posted", "success"),
    ("JRN-2026-0835", "10 Nov 2026", "Draf koreksi depresiasi mesin produksi lini 2", "IDR @ 1,00", "28.500.000", "28.500.000", "Draf", "neutral"),
]


def build_journals_workspace(b: DocBuilder, artboard: str, t: dict[str, Any], selected_idx: int | None = 0) -> None:
    add_navbar(b, artboard, t)
    add_sidebar(b, artboard, t, active_label="Buku Besar & Jurnal")

    main = b.frame(
        "MainContent",
        artboard,
        256,
        56,
        1184,
        884,
        {
            "fill": t["canvas"],
            "layout": "vertical",
            "gap": 14,
            "padding": [20, 24, 20, 24],
            "clip": True,
        },
        bind_var=("fill", "bg/canvas"),
    )

    # Header
    hdr = b.frame("PageHeader", main, 0, 0, 1136, 52, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    hl = b.frame("HL", hdr, 0, 0, 620, 50, {"fill": "none", "layout": "vertical", "gap": 3})
    b.text("Keuangan & Akuntansi  /  Buku Besar & Jurnal", hl, size=11, color=t["ink_muted"], style="SemiBold")
    b.text("Buku Besar & Jurnal", hl, size=21, color=t["ink"], style="Bold", letter_spacing="-2%", text_style="Display/H1")

    hr = b.frame("HR", hdr, 0, 0, 320, 36, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 8})
    add_button(b, hr, "Ekspor CSV", t, variant="outline", size="sm", icon_name="download")
    add_button(b, hr, "Buat Entri Jurnal", t, variant="primary", size="sm", icon_name="plus")

    # 4 Compact Metric Strip Cards
    strip = b.frame("JournalKpiStrip", main, 0, 0, 1136, 78, {"fill": "none", "layout": "horizontal", "gap": 14})
    strip_items = [
        ("ENTRI NOVEMBER", "148 Jurnal", "+12 minggu ini", "info"),
        ("MENUNGGU POSTING", "6 Draf", "Perlu otorisasi", "warning"),
        ("MUTASI DEBIT PERIODE", "Rp 18,64 M", "Multi-currency ADR-014", "brand"),
        ("KESEIMBANGAN GL", "100% Seimbang", "Selisih Rp 0,00", "success"),
    ]
    for k_lbl, k_val, k_sub, k_tone in strip_items:
        sc = b.frame(
            f"Strip/{k_lbl}",
            strip,
            0,
            0,
            273.5,
            78,
            {
                "fill": t["surface"],
                "stroke": t["border"],
                "strokeWeight": 1,
                "cornerRadius": 10,
                "layout": "vertical",
                "mainAlign": "spaceBetween",
                "padding": [12, 14, 12, 14],
            },
            shadow=t["shadow_card"],
        )
        b.text(k_lbl, sc, size=10, color=t["ink_muted"], style="Bold", letter_spacing="4%")
        bot = b.frame("R", sc, 0, 0, 245, 26, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
        b.text(k_val, bot, size=17, color=t["ink"], style="Bold", tnum=True, letter_spacing="-2%")
        add_badge(b, bot, k_sub, k_tone, t, dot=False, compact=True)

    # Tabs + Filter Bar Card
    panel = b.frame(
        "JournalTableCard",
        main,
        0,
        0,
        1136,
        662,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 12,
            "layout": "vertical",
            "clip": True,
        },
        shadow=t["shadow_card"],
    )

    # Sub-tabs + Filter row
    bar = b.frame(
        "Toolbar",
        panel,
        0,
        0,
        1136,
        92,
        {
            "fill": "none",
            "stroke": t["border"],
            "strokeBottomWeight": 1,
            "layout": "vertical",
        },
    )
    tabs = b.frame(
        "SubTabs",
        bar,
        0,
        0,
        1136,
        40,
        {
            "fill": "none",
            "stroke": t["border"],
            "strokeBottomWeight": 1,
            "layout": "horizontal",
            "crossAlign": "center",
            "gap": 4,
            "padding": [0, 16, 0, 16],
        },
    )
    for t_lbl, is_act in [("Daftar Jurnal (148)", True), ("Neraca Saldo", False), ("Jurnal Penutup", False)]:
        tb = b.frame(
            f"Tab/{t_lbl}",
            tabs,
            0,
            0,
            152 if is_act else 120,
            40,
            {
                "fill": "none",
                "stroke": t["brand_primary"] if is_act else "none",
                "strokeBottomWeight": 2 if is_act else 0,
                "layout": "horizontal",
                "mainAlign": "center",
                "crossAlign": "center",
            },
        )
        b.text(t_lbl, tb, size=12, color=t["brand_primary"] if is_act else t["ink_muted"], style="SemiBold" if is_act else "Regular")

    flt = b.frame(
        "FiltersRow",
        bar,
        0,
        0,
        1136,
        52,
        {
            "fill": "none",
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 16, 0, 16],
        },
    )
    flt_l = b.frame("FL", flt, 0, 0, 740, 34, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8})
    add_input_field(b, flt_l, None, "Cari nomor jurnal, memo, kode akun...", t, width=310, input_h=32, left_icon="search", state="placeholder")
    add_input_field(b, flt_l, None, "Status: Semua", t, width=154, input_h=32, left_icon="filter", right_icon="chevron-down")
    add_input_field(b, flt_l, None, "Valas: Semua", t, width=136, input_h=32, right_icon="chevron-down")
    add_input_field(b, flt, None, "Nov 2026", t, width=132, input_h=32, left_icon="calendar", right_icon="chevron-down")

    # 7-Column Table Header (1136px total)
    cols = [
        ("NOMOR JURNAL", 148, "left"),
        ("TGL POSTING", 108, "left"),
        ("REFERENSI / MEMO", 322, "left"),
        ("VALAS & KURS", 124, "left"),
        ("TOTAL DEBIT (IDR)", 152, "right"),
        ("TOTAL KREDIT (IDR)", 152, "right"),
        ("STATUS", 130, "center"),
    ]
    thead = b.frame(
        "THead",
        panel,
        0,
        0,
        1136,
        38,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeBottomWeight": 1,
            "layout": "horizontal",
            "crossAlign": "center",
        },
    )
    for clbl, cw, calign in cols:
        cell = b.frame(
            f"TH/{clbl}",
            thead,
            0,
            0,
            cw,
            38,
            {
                "fill": "none",
                "layout": "horizontal",
                "crossAlign": "center",
                "mainAlign": "max" if calign == "right" else ("center" if calign == "center" else "min"),
                "padding": [0, 12, 0, 12],
            },
        )
        b.text(clbl, cell, size=10, color=t["ink_muted"], style="Bold", letter_spacing="4%")

    # 8 Data Rows (60px each = 480px)
    tbody = b.frame("TBody", panel, 0, 0, 1136, 480, {"fill": "none", "layout": "vertical"})
    for idx, (no, dt, memo, fx, deb, crd, st, st_tone) in enumerate(JOURNAL_ROWS):
        is_sel = selected_idx == idx
        row_bg = t["surface_selected"] if is_sel else (t["surface_alt"] if idx % 2 == 1 else t["surface"])
        tr = b.frame(
            f"TR/{no}",
            tbody,
            0,
            0,
            1136,
            60,
            {
                "fill": row_bg,
                "stroke": t["border_subtle"],
                "strokeBottomWeight": 1,
                "layout": "horizontal",
                "crossAlign": "center",
            },
        )
        # Col 1: Journal No + Active marker
        c1 = b.frame("C1", tr, 0, 0, 148, 60, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 7, "padding": [0, 12, 0, 12]})
        if is_sel:
            b.rect("SelBar", c1, 0, 0, 3, 24, {"fill": t["brand_primary"], "cornerRadius": 2})
        b.text(no, c1, size=11.5, color=t["brand_primary"] if is_sel else t["ink"], style="Bold", family="Menlo", zero=True)

        # Col 2: Posting Date
        c2 = b.frame("C2", tr, 0, 0, 108, 60, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "padding": [0, 12, 0, 12]})
        b.text(dt, c2, size=11.5, color=t["ink_muted"], tnum=True)

        # Col 3: Memo
        c3 = b.frame("C3", tr, 0, 0, 322, 60, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "padding": [0, 12, 0, 12]})
        b.text(memo, c3, size=12, color=t["ink_secondary"], width=298, truncate=True)

        # Col 4: FX
        c4 = b.frame("C4", tr, 0, 0, 124, 60, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "padding": [0, 12, 0, 12]})
        fx_col = t["brand_soft_ink"] if not fx.startswith("IDR") else t["ink_muted"]
        b.text(fx, c4, size=11, color=fx_col, style="SemiBold" if not fx.startswith("IDR") else "Regular", family="Menlo", zero=True)

        # Col 5: Debit
        c5 = b.frame("C5", tr, 0, 0, 152, 60, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "padding": [0, 12, 0, 12]})
        b.text(deb, c5, size=11.5, color=t["ink"], style="Bold", family="Menlo", tnum=True, zero=True)

        # Col 6: Credit
        c6 = b.frame("C6", tr, 0, 0, 152, 60, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "padding": [0, 12, 0, 12]})
        b.text(crd, c6, size=11.5, color=t["ink"], style="Bold", family="Menlo", tnum=True, zero=True)

        # Col 7: StatusBadge
        c7 = b.frame("C7", tr, 0, 0, 130, 60, {"fill": "none", "layout": "horizontal", "mainAlign": "center", "crossAlign": "center", "padding": [0, 10, 0, 10]})
        add_badge(b, c7, st, st_tone, t, dot=True, compact=True)

    # Pagination Footer
    tfoot = b.frame(
        "TFoot",
        panel,
        0,
        0,
        1136,
        52,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeTopWeight": 1,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 16, 0, 16],
        },
    )
    b.text("Menampilkan 1–8 dari 148 entri jurnal · Seluruh buku besar seimbang", tfoot, size=11.5, color=t["ink_muted"])
    pag = b.frame("Pages", tfoot, 0, 0, 224, 28, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 4})
    p_prev = b.frame("Prev", pag, 0, 0, 28, 28, {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 6, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
    b.icon("chevron-left", p_prev, size=13, color=t["ink_muted"])
    for pg_lbl, pg_act in [("1", True), ("2", False), ("3", False), ("…", False), ("19", False)]:
        pb = b.frame(
            f"Pg/{pg_lbl}",
            pag,
            0,
            0,
            28,
            28,
            {
                "fill": t["brand_primary"] if pg_act else t["surface"],
                "stroke": "none" if pg_act else t["border"],
                "strokeWeight": 0 if pg_act else 1,
                "cornerRadius": 6,
                "layout": "horizontal",
                "mainAlign": "center",
                "crossAlign": "center",
            },
        )
        b.text(pg_lbl, pb, size=11, color="#FFFFFF" if pg_act else t["ink_secondary"], style="Bold" if pg_act else "Regular", family="Menlo")
    p_next = b.frame("Next", pag, 0, 0, 28, 28, {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 6, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
    b.icon("chevron-right", p_next, size=13, color=t["ink_secondary"])


# ==============================================================================
# 9. SCREEN 01 & 05: DASHBOARD EKSEKUTIF (LIGHT & DARK)
# ==============================================================================


def build_dashboard_artboard(
    b: DocBuilder,
    artboard: str,
    t: dict[str, Any],
) -> None:
    add_navbar(b, artboard, t)
    add_sidebar(b, artboard, t, active_label="Dashboard Eksekutif")

    main = b.frame(
        "MainContent",
        artboard,
        256,
        56,
        1184,
        884,
        {
            "fill": t["canvas"],
            "layout": "vertical",
            "gap": 14,
            "padding": [20, 24, 20, 24],
            "clip": True,
        },
        bind_var=("fill", "bg/canvas"),
    )

    # 1. Header Row (48px)
    hdr = b.frame(
        "DashboardHeader",
        main,
        0,
        0,
        1136,
        48,
        {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"},
    )
    hl = b.frame("HL", hdr, 0, 0, 620, 48, {"fill": "none", "layout": "vertical", "gap": 3})
    b.text(
        "Dashboard Eksekutif",
        hl,
        size=21,
        color=t["ink"],
        style="Bold",
        letter_spacing="-2%",
        text_style="Display/H1",
    )
    b.text(
        "Ikhtisar keuangan konsolidasi & persetujuan operasional · PT Sinergi Nusantara · Nov 2026",
        hl,
        size=12,
        color=t["ink_muted"],
    )

    hr = b.frame("HR", hdr, 0, 0, 430, 36, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 8})
    per_btn = b.frame(
        "PeriodPill",
        hr,
        0,
        0,
        142,
        34,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 8,
            "layout": "horizontal",
            "mainAlign": "center",
            "crossAlign": "center",
            "gap": 7,
            "padding": [0, 12, 0, 12],
        },
    )
    b.icon("calendar", per_btn, size=13.5, color=t["brand_primary"])
    b.text("Periode: Nov 2026", per_btn, size=11.5, color=t["ink"], style="SemiBold")
    add_button(b, hr, "Segarkan", t, variant="outline", size="sm", icon_name="refresh-cw")
    add_button(b, hr, "Buat Jurnal", t, variant="primary", size="sm", icon_name="plus")

    # 2. 4 KPI Cards (112px)
    kpi_row = b.frame("KpiGrid", main, 0, 0, 1136, 112, {"fill": "none", "layout": "horizontal", "gap": 16})
    add_kpi_card(
        b,
        kpi_row,
        "TOTAL ASET GL",
        "Rp 48,25 M",
        "vs Okt (Rp 45,35 M)",
        "+6,4%",
        "success",
        "brand",
        "landmark",
        t,
    )
    add_kpi_card(
        b,
        kpi_row,
        "KEWAJIBAN & AP",
        "Rp 14,82 M",
        "Ekuitas Rp 33,43 M",
        "6 Jatuh Tempo",
        "warning",
        "warning",
        "arrow-down-left",
        t,
    )
    add_kpi_card(
        b,
        kpi_row,
        "PENDAPATAN & AR",
        "Rp 29,40 M",
        "Margin bersih 34,8%",
        "+14,2%",
        "success",
        "success",
        "trending-up",
        t,
    )
    add_kpi_card(
        b,
        kpi_row,
        "KAS, BANK & VALAS",
        "Rp 12,18 M",
        "USD 767,2K @ 15.875",
        "4 Rekening",
        "brand",
        "sky",
        "coins",
        t,
    )

    # 3. 2 Honest Analytical Charts (286px)
    charts = b.frame("AnalyticsGrid", main, 0, 0, 1136, 286, {"fill": "none", "layout": "horizontal", "gap": 16})
    add_gl_bar_chart(b, charts, t, width=560, height=286)
    add_sales_area_chart(b, charts, t, width=560, height=286)

    # 4. Bottom 2:1 Workspace Grid (DashboardWorkspace.tsx: Approval Queue + Quick Actions & BI Sync)
    bottom = b.frame("BottomGrid", main, 0, 0, 1136, 354, {"fill": "none", "layout": "horizontal", "gap": 16})

    # Left Col (2/3 width = 752px): Antrean Persetujuan Workflow Tertunda
    queue = b.frame(
        "ApprovalQueueCard",
        bottom,
        0,
        0,
        752,
        354,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 12,
            "layout": "vertical",
            "clip": True,
        },
        bind_var=("fill", "bg/surface"),
        shadow=t["shadow_card"],
    )
    q_hdr = b.frame(
        "QHeader",
        queue,
        0,
        0,
        752,
        54,
        {
            "fill": "none",
            "stroke": t["border"],
            "strokeBottomWeight": 1,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 18, 0, 18],
        },
    )
    q_hl = b.frame("QHL", q_hdr, 0, 0, 460, 28, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10})
    b.text("Antrean Persetujuan Workflow Tertunda", q_hl, size=14, color=t["ink"], style="SemiBold")
    add_badge(b, q_hl, "3 Menunggu", "warning", t, dot=True, compact=True)
    q_all = b.frame("AllLink", q_hdr, 0, 0, 110, 24, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 4})
    b.text("Buka Antrean", q_all, size=11.5, color=t["brand_primary"], style="SemiBold")
    b.icon("chevron-right", q_all, size=13, color=t["brand_primary"])

    approvals = [
        (
            "PR-2026-0419",
            "Pengadaan",
            "info",
            "shopping-cart",
            "Pengadaan Server Rack & Lisensi Storage DC-JKT",
            "Diajukan oleh Divisi Infrastruktur TI · 18 menit lalu",
            "Rp 245.000.000",
            "Setujui",
            "primary",
        ),
        (
            "JRN-2026-0839",
            "Buku Besar",
            "brand",
            "book-open",
            "Revaluasi Kurs Tengah Akhir Bulan Posisi USD/SGD",
            "Diajukan oleh Rina Hartati (Akunting) · 1 jam lalu",
            "Rp 84.320.000",
            "Setujui",
            "primary",
        ),
        (
            "PAY-2026-11",
            "SDM & Payroll",
            "warning",
            "wallet",
            "Otorisasi Payroll Batch November · 142 Karyawan",
            "Perlu reviu CFO sebelum rilis transfer · 3 jam lalu",
            "Rp 1.482.600.000",
            "Tinjau",
            "secondary",
        ),
    ]
    q_body = b.frame("QList", queue, 0, 0, 752, 300, {"fill": "none", "layout": "vertical", "padding": [8, 16, 12, 16], "gap": 8})
    for idx, (doc_no, mod_lbl, mod_tone, ico_name, title, meta, amt, cta_lbl, cta_var) in enumerate(approvals):
        ic_bg, ic_bdr, ic_fg = t["kpi_icon"]["brand" if mod_tone == "info" else ("indigo" if mod_tone == "brand" else "warning")]
        row = b.frame(
            f"ApprovalRow/{doc_no}",
            q_body,
            0,
            0,
            720,
            86,
            {
                "fill": t["surface_alt"] if idx == 0 else t["surface"],
                "stroke": t["border"],
                "strokeWeight": 1,
                "cornerRadius": 10,
                "layout": "horizontal",
                "mainAlign": "spaceBetween",
                "crossAlign": "center",
                "padding": [12, 14, 12, 14],
            },
        )
        rl = b.frame("Left", row, 0, 0, 412, 58, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 12})
        ic_box = b.frame(
            "Ic",
            rl,
            0,
            0,
            36,
            36,
            {"fill": ic_bg, "stroke": ic_bdr, "strokeWeight": 1, "cornerRadius": 9, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"},
        )
        b.icon(ico_name, ic_box, size=16, color=ic_fg)

        info_col = b.frame("Info", rl, 0, 0, 360, 56, {"fill": "none", "layout": "vertical", "gap": 4})
        top_tags = b.frame("Tags", info_col, 0, 0, 340, 20, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8})
        b.text(doc_no, top_tags, size=11.5, color=t["ink"], style="Bold", family="Menlo", zero=True)
        add_badge(b, top_tags, mod_lbl, mod_tone, t, dot=False, compact=True)
        b.text(title, info_col, size=12.5, color=t["ink"], style="SemiBold", truncate=True, width=356)
        b.text(meta, info_col, size=11, color=t["ink_muted"], truncate=True, width=356)

        rr = b.frame("Right", row, 0, 0, 272, 44, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 12})
        amt_box = b.frame("Amt", rr, 0, 0, 126, 36, {"fill": "none", "layout": "vertical", "crossAlign": "max", "gap": 2})
        b.text("Nilai Transaksi", amt_box, size=10, color=t["ink_muted"])
        b.text(amt, amt_box, size=12, color=t["ink"], style="Bold", family="Menlo", tnum=True, zero=True, align="right")
        acts = b.frame("Acts", rr, 0, 0, 134, 30, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 6})
        add_button(b, acts, "Detail", t, variant="outline", size="xs")
        add_button(b, acts, cta_lbl, t, variant=cta_var, size="xs")

    # Right Col (1/3 width = 368px): Aksi Cepat & Sinkronisasi BI
    aside = b.frame(
        "QuickActionsAndSyncCard",
        bottom,
        0,
        0,
        368,
        354,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 12,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "padding": [16, 16, 14, 16],
        },
        bind_var=("fill", "bg/surface"),
        shadow=t["shadow_card"],
    )
    a_top = b.frame("Top", aside, 0, 0, 336, 190, {"fill": "none", "layout": "vertical", "gap": 10})
    b.text("Aksi Cepat & Pintasan Modul", a_top, size=14, color=t["ink"], style="SemiBold")
    qa_list = b.frame("QAList", a_top, 0, 0, 336, 160, {"fill": "none", "layout": "vertical", "gap": 6})
    qa_items = [
        ("Entri Jurnal GL", "Posting multi-valas ADR-014", "book-open", "brand"),
        ("Faktur Pajak & e-Faktur", "PPN Masa & PPh 23", "receipt", "sky"),
        ("Kurs Tengah BI", "USD 15.875 · SGD 11.840", "coins", "warning"),
        ("Laporan BI & Ekspor", "Neraca, Laba Rugi, Arus Kas", "bar-chart-3", "success"),
    ]
    for qlbl, qsub, qico, qacc in qa_items:
        qbg, qbdr, qfg = t["kpi_icon"][qacc]
        row_btn = b.frame(
            f"QA/{qlbl}",
            qa_list,
            0,
            0,
            336,
            35,
            {
                "fill": t["surface_alt"],
                "stroke": t["border"],
                "strokeWeight": 1,
                "cornerRadius": 8,
                "layout": "horizontal",
                "mainAlign": "spaceBetween",
                "crossAlign": "center",
                "padding": [0, 10, 0, 8],
            },
        )
        ql = b.frame("L", row_btn, 0, 0, 294, 24, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 9})
        ic_pill = b.frame(
            "Ic",
            ql,
            0,
            0,
            23,
            23,
            {"fill": qbg, "stroke": qbdr, "strokeWeight": 1, "cornerRadius": 6, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"},
        )
        b.icon(qico, ic_pill, size=12.5, color=qfg)
        qtx = b.frame("Tx", ql, 0, 0, 260, 22, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 5})
        b.text(qlbl, qtx, size=11.5, color=t["ink"], style="SemiBold")
        b.text(f"· {qsub}", qtx, size=10, color=t["ink_muted"])
        b.icon("chevron-right", row_btn, size=13, color=t["ink_muted"])

    # Sync Status Inset
    sync_box = b.frame(
        "BISyncBox",
        aside,
        0,
        0,
        336,
        122,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 10,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "padding": [12, 13, 12, 13],
        },
    )
    sh = b.frame("SH", sync_box, 0, 0, 310, 20, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Status Sinkronisasi & Outbox", sh, size=11.5, color=t["ink"], style="SemiBold")
    add_badge(b, sh, "Tersinkron", "success", t, dot=True, compact=True)
    for k, v in [
        ("Cortex Dataform BI", "2 menit lalu · 100%"),
        ("Transactional Outbox", "0 antrean tertunda"),
        ("Kurs Spot BI · USD/IDR", "15.875,00 (Valid)"),
    ]:
        sr = b.frame("SR", sync_box, 0, 0, 310, 18, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
        b.text(k, sr, size=11, color=t["ink_muted"])
        b.text(v, sr, size=11, color=t["ink_secondary"], style="SemiBold", family="Menlo", zero=True)


# ==============================================================================
# 10. SCREEN 03: SIDEDRAWER (XL) — DETAIL JURNAL
# ==============================================================================


def build_drawer_artboard(b: DocBuilder, artboard: str, t: dict[str, Any]) -> None:
    build_journals_workspace(b, artboard, t, selected_idx=0)

    # Scrim overlay covering full 1440x940 workspace
    b.rect("Scrim", artboard, 0, 0, 1440, 940, {"fill": t["scrim"]})

    # 672px XL SideDrawer (JournalDetailDrawer.tsx)
    drawer = b.frame(
        "SideDrawer/JournalDetail",
        artboard,
        768,
        0,
        672,
        940,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeLeftWeight": 1,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "clip": True,
        },
        shadow=t["shadow_drawer"],
    )

    # Header (74px)
    dhdr = b.frame(
        "DrawerHeader",
        drawer,
        0,
        0,
        672,
        74,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeBottomWeight": 1,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 22, 0, 22],
        },
    )
    dhl = b.frame("DHL", dhdr, 0, 0, 480, 44, {"fill": "none", "layout": "vertical", "gap": 3})
    dhl_top = b.frame("Top", dhl, 0, 0, 420, 24, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10})
    b.text("Detail Entri Jurnal", dhl_top, size=16.5, color=t["ink"], style="Bold")
    add_badge(b, dhl_top, "Posted", "success", t, dot=True, compact=True)
    b.text("JRN-2026-0842 · PT-SN · Multi-Currency ADR-014", dhl, size=11.5, color=t["ink_muted"], family="Menlo", zero=True)

    close_btn = b.frame(
        "CloseBtn",
        dhdr,
        0,
        0,
        32,
        32,
        {"fill": t["surface_muted"], "cornerRadius": 8, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"},
    )
    b.icon("x", close_btn, size=15, color=t["ink_secondary"])

    # Body (802px)
    dbody = b.frame(
        "DrawerBody",
        drawer,
        0,
        0,
        672,
        802,
        {"fill": t["surface"], "layout": "vertical", "gap": 16, "padding": [18, 22, 18, 22], "clip": True},
    )

    # 3x2 Metadata Card
    meta_card = b.frame(
        "MetaGridCard",
        dbody,
        0,
        0,
        628,
        122,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 10,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "padding": [14, 16, 14, 16],
        },
    )
    meta_rows = [
        [("PERIODE FISKAL", "2026-11 · Terbuka"), ("TGL POSTING", "15 Nov 2026"), ("SUMBER MODUL", "AP Settlement")],
        [("ENTITAS / CABANG", "PT-SN · Jakarta Pusat"), ("DIBUAT OLEH", "Rina Hartati"), ("REF EKSTERNAL", "INV-VND-1192")],
    ]
    for mr in meta_rows:
        row_f = b.frame("MetaRow", meta_card, 0, 0, 596, 40, {"fill": "none", "layout": "horizontal", "gap": 12})
        for mlbl, mval in mr:
            mc = b.frame("MC", row_f, 0, 0, 190, 40, {"fill": "none", "layout": "vertical", "gap": 3})
            b.text(mlbl, mc, size=9.5, color=t["ink_muted"], style="Bold", letter_spacing="4%")
            is_code = "2026" in mval or "INV-" in mval or "PT-SN" in mval
            b.text(mval, mc, size=12, color=t["ink"], style="SemiBold", family="Menlo" if is_code else "Inter", zero=is_code)

    # Memo Box
    memo_box = b.frame(
        "MemoBox",
        dbody,
        0,
        0,
        628,
        64,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 10,
            "layout": "vertical",
            "gap": 4,
            "padding": [11, 14, 11, 14],
        },
    )
    b.text("MEMO & DESKRIPSI TRANSAKSI", memo_box, size=9.5, color=t["ink_muted"], style="Bold", letter_spacing="4%")
    b.text("Pelunasan invoice pengadaan lisensi cloud & infrastruktur Q4 — Vendor Singapura (Spot BCA)", memo_box, size=12.5, color=t["ink"])

    # 6-Column Account Lines Table
    lines_sec = b.frame("LinesSection", dbody, 0, 0, 628, 304, {"fill": "none", "layout": "vertical", "gap": 8})
    ls_hdr = b.frame("LSH", lines_sec, 0, 0, 628, 22, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Baris Akun & Nilai Fungsional", ls_hdr, size=13, color=t["ink"], style="SemiBold")
    b.text("3 baris · Terposting ke GL", ls_hdr, size=11, color=t["ink_muted"])

    ltbl = b.frame(
        "LinesTable",
        lines_sec,
        0,
        0,
        628,
        270,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "clip": True},
    )
    l_cols = [
        ("KODE", 78, "left"),
        ("NAMA AKUN", 174, "left"),
        ("VALAS", 58, "center"),
        ("DEBIT", 96, "right"),
        ("KREDIT", 96, "right"),
        ("FUNGSIONAL (IDR)", 126, "right"),
    ]
    lth = b.frame(
        "LTH",
        ltbl,
        0,
        0,
        628,
        36,
        {"fill": t["surface_alt"], "stroke": t["border"], "strokeBottomWeight": 1, "layout": "horizontal", "crossAlign": "center"},
    )
    for cl, cw, ca in l_cols:
        cf = b.frame(f"LTH/{cl}", lth, 0, 0, cw, 36, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "mainAlign": "max" if ca == "right" else ("center" if ca == "center" else "min"), "padding": [0, 9, 0, 9]})
        b.text(cl, cf, size=9.5, color=t["ink_muted"], style="Bold", letter_spacing="4%")

    detail_lines = [
        ("2101-02", "Hutang Dagang Impor", "USD", "12.000,00", "0,00", "Rp 190.500.000"),
        ("1103-01", "Bank BCA Valas USD", "USD", "0,00", "11.980,00", "Rp 190.182.500"),
        ("6104-01", "Selisih Kurs Terealisasi", "IDR", "0,00", "317.500,00", "Rp 317.500"),
    ]
    for code, acc_name, ccy, deb, crd, func in detail_lines:
        lr = b.frame(
            f"DL/{code}",
            ltbl,
            0,
            0,
            628,
            60,
            {"fill": t["surface"], "stroke": t["border_subtle"], "strokeBottomWeight": 1, "layout": "horizontal", "crossAlign": "center"},
        )
        for val, cw, ca, fam, col, st in [
            (code, 78, "left", "Menlo", t["brand_primary"], "Bold"),
            (acc_name, 174, "left", "Inter", t["ink"], "SemiBold"),
            (ccy, 58, "center", "Menlo", t["ink_muted"], "Bold"),
            (deb, 96, "right", "Menlo", t["ink"] if deb != "0,00" else t["ink_subtle"], "Bold" if deb != "0,00" else "Regular"),
            (crd, 96, "right", "Menlo", t["ink"] if crd != "0,00" else t["ink_subtle"], "Bold" if crd != "0,00" else "Regular"),
            (func, 126, "right", "Menlo", t["ink"], "Bold"),
        ]:
            lc = b.frame("LC", lr, 0, 0, cw, 60, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "mainAlign": "max" if ca == "right" else ("center" if ca == "center" else "min"), "padding": [0, 9, 0, 9]})
            b.text(val, lc, size=11, color=col, style=st, family=fam, tnum=True, zero=(fam == "Menlo"), truncate=True, width=cw - 16)

    # Total Bar inside LinesTable
    ltot = b.frame(
        "LTotal",
        ltbl,
        0,
        0,
        628,
        54,
        {"fill": t["brand_soft"], "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 14, 0, 14]},
    )
    b.text("TOTAL FUNGSIONAL (IDR)", ltot, size=11, color=t["brand_soft_ink"], style="Bold")
    ltot_r = b.frame("R", ltot, 0, 0, 386, 26, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 10})
    b.text("D Rp 190.500.000  =  K Rp 190.500.000", ltot_r, size=11, color=t["brand_soft_ink"], style="Bold", family="Menlo", tnum=True)
    add_badge(b, ltot_r, "Seimbang", "success", t, dot=True, compact=True)

    # ADR-014 Tri-Currency Card
    tri = b.frame(
        "TriCurrencyCard",
        dbody,
        0,
        0,
        628,
        92,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 10,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "padding": [12, 14, 12, 14],
        },
    )
    tri_h = b.frame("TH", tri, 0, 0, 600, 18, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Kepatuhan Tri-Currency (ADR-014)", tri_h, size=11.5, color=t["ink"], style="SemiBold")
    b.text("Spot BI @ 15.875,00", tri_h, size=10.5, color=t["ink_muted"], family="Menlo")
    tri_cols = b.frame("TC", tri, 0, 0, 600, 38, {"fill": "none", "layout": "horizontal", "gap": 12})
    for tlbl, tval in [
        ("1. MATA UANG TRANSAKSI", "USD 12.000,00"),
        ("2. FUNGSIONAL ENTITAS", "IDR 190.500.000"),
        ("3. KONSOLIDASI GRUP", "USD 12.000,00"),
    ]:
        tc = b.frame("TCol", tri_cols, 0, 0, 192, 38, {"fill": "none", "layout": "vertical", "gap": 3})
        b.text(tlbl, tc, size=9.5, color=t["ink_muted"], style="Bold")
        b.text(tval, tc, size=12, color=t["ink"], style="Bold", family="Menlo", tnum=True, zero=True)

    # Audit & Outbox Notice
    aud = b.frame(
        "AuditOutboxCard",
        dbody,
        0,
        0,
        628,
        54,
        {
            "fill": t["badges"]["success"][0],
            "stroke": t["badges"]["success"][1],
            "strokeWeight": 1,
            "cornerRadius": 10,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 14, 0, 14],
        },
    )
    aud_l = b.frame("L", aud, 0, 0, 440, 24, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 9})
    b.icon("check-circle-2", aud_l, size=16, color=t["badges"]["success"][3])
    b.text("Transactional Outbox terkirim · Event gl.journal.posted (#88412)", aud_l, size=11.5, color=t["badges"]["success"][3], style="SemiBold")
    b.text("15 Nov · 09:34 WIB", aud, size=10.5, color=t["badges"]["success"][3], family="Menlo")

    # Sticky Footer (64px)
    dfoot = b.frame(
        "DrawerFooter",
        drawer,
        0,
        0,
        672,
        64,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeTopWeight": 1,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 22, 0, 22],
        },
    )
    add_button(b, dfoot, "Tutup", t, variant="ghost", size="sm")
    df_r = b.frame("DFR", dfoot, 0, 0, 280, 36, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 8})
    add_button(b, df_r, "Cetak Bukti", t, variant="outline", size="sm", icon_name="printer")
    add_button(b, df_r, "Balik Jurnal", t, variant="danger-soft", size="sm", icon_name="rotate-ccw")


# ==============================================================================
# 11. SCREEN 04: APPDIALOG (2XL) — BUAT ENTRI JURNAL
# ==============================================================================


def build_dialog_artboard(b: DocBuilder, artboard: str, t: dict[str, Any]) -> None:
    build_journals_workspace(b, artboard, t, selected_idx=None)

    b.rect("Scrim", artboard, 0, 0, 1440, 940, {"fill": t["scrim"]})

    # Centered 896px 2XL Dialog with balanced vertical height (no awkward dead space!)
    dlg = b.frame(
        "AppDialog/CreateJournal",
        artboard,
        272,
        58,
        896,
        824,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 16,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "clip": True,
        },
        shadow=t["shadow_modal"],
    )

    # Header (68px)
    hdr = b.frame(
        "DialogHeader",
        dlg,
        0,
        0,
        896,
        68,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeBottomWeight": 1,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 22, 0, 22],
        },
    )
    hl = b.frame("HL", hdr, 0, 0, 620, 42, {"fill": "none", "layout": "vertical", "gap": 2})
    b.text("Buat Entri Jurnal Umum", hl, size=17, color=t["ink"], style="Bold")
    b.text("Lengkapi header entri & baris seimbang multi-mata uang (ADR-014 Tri-Currency)", hl, size=11.5, color=t["ink_muted"])
    xbtn = b.frame("CloseBtn", hdr, 0, 0, 32, 32, {"fill": t["surface_muted"], "cornerRadius": 8, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
    b.icon("x", xbtn, size=15, color=t["ink_secondary"])

    # Body (692px)
    body = b.frame(
        "DialogBody",
        dlg,
        0,
        0,
        896,
        692,
        {"fill": t["surface"], "layout": "vertical", "gap": 14, "padding": [18, 22, 16, 22]},
    )

    # 2x3 Header Form Fields (each col = 273px, gap = 16 -> 3*273 + 32 = 851px)
    r1 = b.frame("FormRow1", body, 0, 0, 852, 58, {"fill": "none", "layout": "horizontal", "gap": 16})
    add_input_field(b, r1, "Nomor Jurnal (Otomatis)", "JRN-2026-0843", t, width=273, input_h=36, state="readonly", mono=True)
    add_input_field(b, r1, "Tanggal Posting", "15 Nov 2026", t, width=273, input_h=36, right_icon="calendar", required=True)
    add_input_field(b, r1, "Tipe Jurnal", "Jurnal Umum (GJ)", t, width=273, input_h=36, right_icon="chevron-down", required=True)

    r2 = b.frame("FormRow2", body, 0, 0, 852, 58, {"fill": "none", "layout": "horizontal", "gap": 16})
    add_input_field(b, r2, "Periode Fiskal", "2026-11 · Terbuka", t, width=273, input_h=36, right_icon="chevron-down")
    add_input_field(
        b,
        r2,
        "Mata Uang & Kurs Tengah BI",
        "15.875,00",
        t,
        width=273,
        input_h=36,
        currency_chip="USD",
        mono=True,
        right_align=True,
    )
    add_input_field(b, r2, "Referensi Eksternal", "INV-VND-2026-1192", t, width=273, input_h=36, mono=True)

    # Memo Field
    add_input_field(
        b,
        body,
        "Memo / Deskripsi Jurnal",
        "Pelunasan invoice pengadaan lisensi cloud & infrastruktur Q4 — Vendor Singapura",
        t,
        width=852,
        input_h=36,
        required=True,
    )

    # Lines Editor Card
    ed = b.frame(
        "LinesEditorCard",
        body,
        0,
        0,
        852,
        362,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 10,
            "layout": "vertical",
            "gap": 10,
            "padding": [12, 14, 12, 14],
        },
    )
    ed_hdr = b.frame("EdHdr", ed, 0, 0, 824, 28, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    ed_hl = b.frame("HL", ed_hdr, 0, 0, 320, 22, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8})
    b.text("Baris Entri Jurnal", ed_hl, size=13, color=t["ink"], style="SemiBold")
    add_badge(b, ed_hl, "3 Baris", "neutral", t, dot=False, compact=True)
    add_button(b, ed_hdr, "Tambah Baris", t, variant="outline", size="xs", icon_name="plus")

    # Editor Table Headers (total 824px, aligned identically with LineRow padding)
    eh_row = b.frame("EdCols", ed, 0, 0, 824, 18, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8, "padding": [0, 10, 0, 10]})
    for cl, cw, ca in [
        ("AKUN PERKIRAAN *", 216, "left"),
        ("CATATAN BARIS", 136, "left"),
        ("DEBIT (VALAS)", 142, "right"),
        ("KREDIT (VALAS)", 142, "right"),
        ("FUNGSIONAL (IDR)", 100, "right"),
        ("", 32, "center"),
    ]:
        ec = b.frame("EC", eh_row, 0, 0, cw, 18, {"fill": "none", "layout": "horizontal", "mainAlign": "max" if ca == "right" else "min", "crossAlign": "center"})
        if cl:
            b.text(cl, ec, size=9.5, color=t["ink_muted"], style="Bold", letter_spacing="4%")

    # 3 Editable Lines (with MoneyInput currency chips + Trash2 delete icon button)
    line_inputs = [
        ("2101-02 · Hutang Impor", "Pelunasan inv Q4", "USD", "12.000,00", "0,00", "Rp 190.500.000", False),
        ("1103-01 · Bank BCA USD", "Transfer SWIFT", "USD", "0,00", "11.980,00", "Rp 190.182.500", False),
        ("6104-01 · Selisih Kurs", "Selisih spot bank", "IDR", "0,00", "317.500,00", "Rp 317.500", True),
    ]
    for acc_s, note_s, ccy, deb_s, crd_s, func_s, is_focused in line_inputs:
        row_box = b.frame(
            f"LineRow/{acc_s}",
            ed,
            0,
            0,
            824,
            72,
            {
                "fill": t["surface"],
                "stroke": t["border"],
                "strokeWeight": 1,
                "cornerRadius": 9,
                "layout": "horizontal",
                "crossAlign": "center",
                "gap": 8,
                "padding": [0, 10, 0, 10],
            },
        )
        add_input_field(b, row_box, None, acc_s, t, width=216, input_h=34, right_icon="chevron-down")
        add_input_field(b, row_box, None, note_s, t, width=136, input_h=34)
        add_input_field(b, row_box, None, deb_s, t, width=142, input_h=34, currency_chip=ccy, mono=True, right_align=True)
        add_input_field(b, row_box, None, crd_s, t, width=142, input_h=34, currency_chip=ccy, mono=True, right_align=True, state="focus" if is_focused else "default")

        fn_cell = b.frame("FuncCell", row_box, 0, 0, 100, 34, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center"})
        b.text(func_s, fn_cell, size=11, color=t["ink"], style="Bold", family="Menlo", tnum=True)

        del_btn = b.frame(
            "DeleteRowBtn",
            row_box,
            0,
            0,
            32,
            32,
            {"fill": "none", "cornerRadius": 7, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"},
        )
        b.icon("trash-2", del_btn, size=14, color=t["ink_muted"])

    # Validation Notice Banner (Business language — zero internal filename leak!)
    val_bar = b.frame(
        "BalanceValidationBar",
        body,
        0,
        0,
        852,
        42,
        {
            "fill": t["badges"]["success"][0],
            "stroke": t["badges"]["success"][1],
            "strokeWeight": 1,
            "cornerRadius": 9,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 14, 0, 14],
        },
    )
    vl = b.frame("VL", val_bar, 0, 0, 680, 24, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8})
    b.icon("check-circle-2", vl, size=15, color=t["badges"]["success"][3])
    b.text("Validasi keseimbangan lolos: Total Debit Fungsional = Total Kredit Fungsional (selisih Rp 0,00).", vl, size=11.5, color=t["badges"]["success"][3], style="SemiBold")
    b.text("Fixed-Point Verified", val_bar, size=10.5, color=t["badges"]["success"][3], style="Bold", family="Menlo")

    # Footer (64px)
    foot = b.frame(
        "DialogFooter",
        dlg,
        0,
        0,
        896,
        64,
        {
            "fill": t["surface_alt"],
            "stroke": t["border"],
            "strokeTopWeight": 1,
            "layout": "horizontal",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [0, 22, 0, 22],
        },
    )
    fl = b.frame("FL", foot, 0, 0, 470, 32, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10})
    b.icon("scale", fl, size=15.5, color=t["badges"]["success"][3])
    b.text("Debit Rp 190.500.000  =  Kredit Rp 190.500.000", fl, size=11.5, color=t["ink"], style="Bold", family="Menlo", tnum=True)
    add_badge(b, fl, "Seimbang", "success", t, dot=True, compact=True)

    fr = b.frame("FR", foot, 0, 0, 330, 36, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 8})
    add_button(b, fr, "Batal", t, variant="ghost", size="sm")
    add_button(b, fr, "Simpan Draf", t, variant="outline", size="sm")
    add_button(b, fr, "Posting Jurnal", t, variant="primary", size="sm", icon_name="check")


# ==============================================================================
# 12. SCREEN 06: MOBILE VIEWPORT (390px) & INTERACTIVE OVERLAYS
# ==============================================================================


def build_mobile_and_overlays_artboard(b: DocBuilder, artboard: str, t: dict[str, Any]) -> None:
    b.text(
        "Responsive Mobile Viewport (390 × 844) & Interactive App Overlays",
        artboard,
        size=15,
        color=t["ink"],
        style="Bold",
        x=40,
        y=22,
    )
    b.text(
        "MobileBottomNav.tsx · CommandPalette.tsx (⌘K) · NotificationPopover.tsx · OrgSwitcher.tsx · UserProfileMenu.tsx",
        artboard,
        size=11.5,
        color=t["ink_muted"],
        x=40,
        y=43,
    )

    # 1. Left: 390x844 iPhone Frame (x=40, y=72)
    phone = b.frame(
        "MobileDevice/390x844",
        artboard,
        40,
        72,
        390,
        844,
        {
            "fill": t["canvas"],
            "stroke": "#0F172A",
            "strokeWeight": 6,
            "cornerRadius": 36,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "clip": True,
        },
        shadow=t["shadow_modal"],
    )

    # Top Mobile Status Bar + Mobile Header
    m_top = b.frame("MobileTop", phone, 0, 0, 390, 96, {"fill": t["surface"], "stroke": t["border"], "strokeBottomWeight": 1, "layout": "vertical"})
    sb = b.frame("StatusBar", m_top, 0, 0, 390, 38, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [8, 22, 0, 22]})
    b.text("09:41", sb, size=12, color=t["ink"], style="Bold", tnum=True)
    b.rect("DynamicIsland", sb, 0, 0, 88, 20, {"fill": "#0F172A", "cornerRadius": 99})
    b.text("5G  100%", sb, size=10.5, color=t["ink"], style="SemiBold", family="Menlo")

    m_nav = b.frame("MobileNavbar", m_top, 0, 0, 390, 56, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 14, 0, 14]})
    mn_l = b.frame("L", m_nav, 0, 0, 200, 34, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 9})
    b.icon("menu", mn_l, size=18, color=t["ink_secondary"])
    b.brand_prism_mark(mn_l, size=26, dark_surface=False)
    b.text("Enterprise ERP", mn_l, size=13, color=t["ink"], style="Bold")

    mn_r = b.frame("R", m_nav, 0, 0, 100, 32, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 8})
    b.icon("search", mn_r, size=16, color=t["ink_secondary"])
    b.icon("bell", mn_r, size=16, color=t["ink_secondary"])
    mav = b.frame("AV", mn_r, 0, 0, 26, 26, {"fill": t["brand_primary"], "cornerRadius": 6, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
    b.text("AS", mav, size=10, color="#FFFFFF", style="Bold")

    # Mobile Scroll Body
    m_body = b.frame("MobileBody", phone, 0, 0, 390, 672, {"fill": t["canvas"], "layout": "vertical", "gap": 12, "padding": [14, 14, 14, 14], "clip": True})
    mh = b.frame("MH", m_body, 0, 0, 362, 42, {"fill": "none", "layout": "vertical", "gap": 2})
    b.text("Dashboard Eksekutif", mh, size=17, color=t["ink"], style="Bold", letter_spacing="-2%")
    b.text("PT-SN · Periode Nov 2026", mh, size=11, color=t["ink_muted"])

    # 2x2 Compact Mobile KPI Grid
    mk_r1 = b.frame("MKR1", m_body, 0, 0, 362, 94, {"fill": "none", "layout": "horizontal", "gap": 10})
    add_kpi_card(b, mk_r1, "TOTAL ASET GL", "Rp 48,25 M", "vs Okt", "+6,4%", "success", "brand", "landmark", t, width=176, height=94)
    add_kpi_card(b, mk_r1, "KEWAJIBAN AP", "Rp 14,82 M", "Eq 33,4M", "6 Tempo", "warning", "warning", "arrow-down-left", t, width=176, height=94)

    mk_r2 = b.frame("MKR2", m_body, 0, 0, 362, 94, {"fill": "none", "layout": "horizontal", "gap": 10})
    add_kpi_card(b, mk_r2, "PENDAPATAN AR", "Rp 29,40 M", "Margin 34,8%", "+14,2%", "success", "success", "trending-up", t, width=176, height=94)
    add_kpi_card(b, mk_r2, "KAS & VALAS", "Rp 12,18 M", "4 Rek", "USD 767K", "brand", "sky", "coins", t, width=176, height=94)

    # Mini Mobile Area Chart Card (NO padding on absolute-positioned frame!)
    m_chart = b.frame("MobileChart", m_body, 0, 0, 362, 162, {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "clip": True}, shadow=t["shadow_card"])
    b.text("Tren Penjualan (Mei – Nov)", m_chart, size=12.5, color=t["ink"], style="SemiBold", x=14, y=12)
    b.text("Rp 6,95 M", m_chart, size=12.5, color="#0F766E", style="Bold", family="Menlo", x=272, y=12, width=76, align="right")
    m_pts = [(0, 82), (55, 68), (110, 74), (165, 50), (220, 44), (275, 24), (330, 12)]
    m_d = _catmull_rom_to_bezier_path(m_pts)
    m_area = f"{m_d} L 330 96 L 0 96 Z"
    b.vector("MArea", m_chart, m_area, x=16, y=46, closed=True, props={"fills": [{"type": "linearGradient", "stops": ["#0F766E36", "#0F766E00"], "from": [0.5, 0], "to": [0.5, 1]}]})
    b.vector("MLine", m_chart, m_d, x=16, y=46, closed=False, props={"fill": "none", "stroke": "#0F766E", "strokeWeight": 2.2, "strokeCap": "round"})
    for mx, my in m_pts:
        b.ellipse("MDot", m_chart, 16 + mx - 2.5, 46 + my - 2.5, 5, 5, {"fill": "#FFFFFF", "stroke": "#0F766E", "strokeWeight": 1.5})
    for mx, mlbl in [(16, "Mei"), (126, "Jul"), (236, "Sep"), (332, "Nov")]:
        b.text(mlbl, m_chart, size=9.5, color=t["ink_muted"], x=mx - 12, y=144, width=32, align="center")

    # Mobile Pending Approvals Card
    m_app = b.frame("MobileApprovals", m_body, 0, 0, 362, 206, {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "gap": 8, "padding": 12}, shadow=t["shadow_card"])
    mah = b.frame("MAH", m_app, 0, 0, 338, 22, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Persetujuan Tertunda", mah, size=12.5, color=t["ink"], style="SemiBold")
    add_badge(b, mah, "3", "warning", t, dot=True, compact=True)
    for mno, mtit, mamt in [
        ("PR-2026-0419", "Server Rack DC-JKT", "Rp 245.000.000"),
        ("JRN-2026-0839", "Revaluasi Kurs USD", "Rp 84.320.000"),
    ]:
        ar = b.frame("AR", m_app, 0, 0, 338, 68, {"fill": t["surface_alt"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 9, "layout": "vertical", "mainAlign": "spaceBetween", "padding": [9, 11, 9, 11]})
        art = b.frame("T", ar, 0, 0, 316, 18, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
        b.text(mno, art, size=11, color=t["brand_primary"], style="Bold", family="Menlo")
        b.text(mamt, art, size=11, color=t["ink"], style="Bold", family="Menlo")
        arb = b.frame("B", ar, 0, 0, 316, 24, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
        b.text(mtit, arb, size=11.5, color=t["ink_secondary"], truncate=True, width=210)
        add_button(b, arb, "Setujui", t, variant="primary", size="xs")

    # MobileBottomNav (Dashboard, Jurnal, AP/AR, Otorisasi, Menu)
    m_bnav = b.frame(
        "MobileBottomNav",
        phone,
        0,
        0,
        390,
        74,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeTopWeight": 1,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "crossAlign": "center",
            "padding": [6, 8, 8, 8],
        },
    )
    mb_row = b.frame("MBItems", m_bnav, 0, 0, 374, 46, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceEvenly", "crossAlign": "center"})
    for mlbl, mico, mact in [
        ("Dashboard", "layout-dashboard", True),
        ("Jurnal", "book-open", False),
        ("AP/AR", "receipt", False),
        ("Otorisasi", "shield-check", False),
        ("Menu", "menu", False),
    ]:
        it = b.frame(f"BNav/{mlbl}", mb_row, 0, 0, 68, 44, {"fill": "none", "layout": "vertical", "mainAlign": "center", "crossAlign": "center", "gap": 3})
        b.icon(mico, it, size=17, color=t["brand_primary"] if mact else t["ink_muted"], stroke_weight=1.9 if mact else 1.5)
        b.text(mlbl, it, size=10, color=t["brand_primary"] if mact else t["ink_muted"], style="Bold" if mact else "Regular")
    b.rect("HomeIndicator", m_bnav, 0, 0, 128, 4, {"fill": t["border_strong"], "cornerRadius": 99})

    # 2. Center-Top: CommandPalette (Cmd+K) (x=470, y=72, w=540, h=408)
    cp = b.frame(
        "Overlay/CommandPalette",
        artboard,
        470,
        72,
        540,
        408,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 14,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "clip": True,
        },
        shadow=t["shadow_modal"],
    )
    cp_search = b.frame(
        "SearchInput",
        cp,
        0,
        0,
        540,
        52,
        {"fill": t["surface"], "stroke": t["border"], "strokeBottomWeight": 1, "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 16, 0, 16]},
    )
    cps_l = b.frame("L", cp_search, 0, 0, 420, 26, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10})
    b.icon("search", cps_l, size=16, color=t["brand_primary"])
    b.text("jurnal valas", cps_l, size=13.5, color=t["ink"], style="SemiBold")
    esc_k = b.frame("ESC", cp_search, 0, 0, 36, 22, {"fill": t["surface_muted"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 5, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
    b.text("ESC", esc_k, size=9.5, color=t["ink_muted"], style="Bold", family="Menlo")

    cp_list = b.frame("Results", cp, 0, 0, 540, 316, {"fill": t["surface"], "layout": "vertical", "gap": 8, "padding": [12, 12, 12, 12]})
    b.text("AKSI CEPAT & MODUL TERKAIT", cp_list, size=10, color=t["ink_muted"], style="Bold", letter_spacing="4%")
    cmd_items = [
        ("plus", "Buat Entri Jurnal Umum", "Keuangan · Membuka AppDialog Entri Baru", "↵", True),
        ("book-open", "Buku Besar & Jurnal", "Keuangan · 148 entri bulan berjalan", "G J", False),
        ("coins", "Mata Uang & Kurs Tengah FX", "ADR-014 · USD 15.875 / SGD 11.840", "G F", False),
        ("receipt", "Pajak & Rekonsiliasi e-Faktur", "Keuangan · Periode Nov 2026", "G P", False),
        ("download", "Ekspor Neraca Saldo (CSV)", "BI Report · Dataform Terverifikasi", "⌘E", False),
    ]
    for ico, title, sub, hint, act in cmd_items:
        crow = b.frame(
            f"Cmd/{title}",
            cp_list,
            0,
            0,
            516,
            50,
            {
                "fill": t["brand_soft"] if act else "none",
                "stroke": t["brand_soft_border"] if act else "none",
                "strokeWeight": 1 if act else 0,
                "cornerRadius": 9,
                "layout": "horizontal",
                "mainAlign": "spaceBetween",
                "crossAlign": "center",
                "padding": [0, 12, 0, 12],
            },
        )
        crl = b.frame("L", crow, 0, 0, 420, 38, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10})
        icw = b.frame("IW", crl, 0, 0, 30, 30, {"fill": t["surface"] if act else t["surface_muted"], "cornerRadius": 7, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
        b.icon(ico, icw, size=14.5, color=t["brand_primary"] if act else t["ink_secondary"])
        txtc = b.frame("TC", crl, 0, 0, 372, 34, {"fill": "none", "layout": "vertical", "gap": 1})
        b.text(title, txtc, size=12.5, color=t["brand_soft_ink"] if act else t["ink"], style="SemiBold")
        b.text(sub, txtc, size=10.5, color=t["ink_muted"])
        hk = b.frame("HK", crow, 0, 0, 32, 22, {"fill": t["surface"] if act else t["surface_alt"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 5, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
        b.text(hint, hk, size=10, color=t["brand_primary"] if act else t["ink_muted"], style="Bold", family="Menlo")

    cp_foot = b.frame("CPFoot", cp, 0, 0, 540, 40, {"fill": t["surface_alt"], "stroke": t["border"], "strokeTopWeight": 1, "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 16, 0, 16]})
    b.text("↑↓ Navigasi   ·   ↵ Pilih Perintah   ·   Esc Tutup", cp_foot, size=10.5, color=t["ink_muted"], family="Menlo")
    b.text("Portal: Backoffice", cp_foot, size=10.5, color=t["brand_primary"], style="SemiBold")

    # 3. Right-Top: NotificationPopover (x=1036, y=72, w=364, h=408)
    np = b.frame(
        "Overlay/NotificationPopover",
        artboard,
        1036,
        72,
        364,
        408,
        {
            "fill": t["surface"],
            "stroke": t["border"],
            "strokeWeight": 1,
            "cornerRadius": 14,
            "layout": "vertical",
            "mainAlign": "spaceBetween",
            "clip": True,
        },
        shadow=t["shadow_popover"],
    )
    nph = b.frame("NPHeader", np, 0, 0, 364, 48, {"fill": t["surface"], "stroke": t["border"], "strokeBottomWeight": 1, "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 14, 0, 14]})
    nph_l = b.frame("L", nph, 0, 0, 160, 22, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 7})
    b.text("Notifikasi", nph_l, size=13.5, color=t["ink"], style="Bold")
    add_badge(b, nph_l, "3 Baru", "danger", t, dot=False, compact=True)
    b.text("Tandai semua dibaca", nph, size=11, color=t["brand_primary"], style="SemiBold")

    np_list = b.frame("NPList", np, 0, 0, 364, 318, {"fill": t["surface"], "layout": "vertical"})
    notifs = [
        ("clock", "info", "PR-2026-0419 menunggu otorisasi", "Nilai Rp 245,0M membutuhkan otorisasi tier-2.", "2 mnt", True),
        ("alert-triangle", "warning", "6 Faktur AP jatuh tempo 48 jam", "Total kewajiban PT Wijaya Steel Rp 312,4M.", "18 mnt", True),
        ("info", "brand", "Kurs Tengah BI diperbarui", "Spot USD/IDR = 15.875,00 · SGD = 11.840,00.", "1 jam", True),
        ("check-circle-2", "success", "Dataform BI Sync selesai 100%", "14 model keuangan & KPI dashboard diperbarui.", "3 jam", False),
    ]
    for nico, ntone, ntit, nsub, ntime, unread in notifs:
        nbg = t["brand_soft"] if unread else t["surface"]
        nc = b.frame(
            f"Notif/{ntit}",
            np_list,
            0,
            0,
            364,
            78,
            {"fill": nbg, "stroke": t["border_subtle"], "strokeBottomWeight": 1, "layout": "horizontal", "crossAlign": "min", "gap": 10, "padding": [12, 14, 12, 14]},
        )
        ic_bg, ic_bdr, _, ic_fg = t["badges"][ntone]
        nib = b.frame("NIc", nc, 0, 0, 28, 28, {"fill": ic_bg, "stroke": ic_bdr, "strokeWeight": 1, "cornerRadius": 99, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
        b.icon(nico, nib, size=13.5, color=ic_fg)
        n_txt = b.frame("NTxt", nc, 0, 0, 294, 54, {"fill": "none", "layout": "vertical", "gap": 2})
        nr1 = b.frame("R1", n_txt, 0, 0, 294, 17, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "gap": 8})
        b.text(ntit, nr1, size=11.5, color=t["ink"], style="SemiBold", truncate=True, width=218)
        nr_meta = b.frame("Meta", nr1, 0, 0, 62, 16, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 5})
        b.text(ntime, nr_meta, size=10, color=t["ink_muted"], family="Menlo", align="right")
        if unread:
            b.ellipse("UnreadDot", nr_meta, 0, 0, 6.5, 6.5, {"fill": t["brand_primary"]})
        b.text(nsub, n_txt, size=11, color=t["ink_secondary"], width=294)

    npf = b.frame("NPFoot", np, 0, 0, 364, 40, {"fill": t["surface_alt"], "stroke": t["border"], "strokeTopWeight": 1, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
    b.text("Lihat Seluruh Riwayat Notifikasi", npf, size=11.5, color=t["brand_primary"], style="SemiBold")

    # 4. Center-Bottom: OrgSwitcher Popover (x=470, y=506, w=450, h=298)
    osw = b.frame(
        "Overlay/OrgSwitcher",
        artboard,
        470,
        506,
        450,
        298,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 14, "layout": "vertical", "padding": [14, 16, 14, 16], "gap": 10},
        shadow=t["shadow_popover"],
    )
    b.text("Organisasi & RLS Scope Aktif", osw, size=13.5, color=t["ink"], style="Bold")
    b.text("Pemilihan entitas mengaktifkan Row-Level Security PostgreSQL per sesi.", osw, size=11, color=t["ink_muted"])
    orgs = [
        ("PT-SN", "PT Sinergi Nusantara (Induk)", "Fungsional: IDR · Jakarta Pusat", True),
        ("PT-SGM", "PT Sinergi Manufaktur", "Fungsional: IDR · Kawasan Cikarang", False),
        ("SG-HOLD", "Sinergi Holding Pte Ltd", "Fungsional: USD · Konsolidasi ASEAN", False),
    ]
    for ocode, oname, ometa, oact in orgs:
        orow = b.frame(
            f"Org/{ocode}",
            osw,
            0,
            0,
            418,
            62,
            {
                "fill": t["brand_soft"] if oact else t["surface_alt"],
                "stroke": t["brand_primary"] if oact else t["border"],
                "strokeWeight": 1.2 if oact else 1,
                "cornerRadius": 10,
                "layout": "horizontal",
                "mainAlign": "spaceBetween",
                "crossAlign": "center",
                "padding": [0, 12, 0, 12],
            },
        )
        ol = b.frame("L", orow, 0, 0, 340, 40, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10})
        ob = b.frame("Badge", ol, 0, 0, 34, 34, {"fill": t["brand_primary"] if oact else t["surface"], "stroke": t["border"] if not oact else "none", "strokeWeight": 1, "cornerRadius": 8, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
        b.icon("building-2", ob, size=15, color="#FFFFFF" if oact else t["ink_secondary"])
        otx = b.frame("Tx", ol, 0, 0, 290, 36, {"fill": "none", "layout": "vertical", "gap": 2})
        b.text(f"{ocode} · {oname}", otx, size=12, color=t["ink"], style="SemiBold")
        b.text(ometa, otx, size=10.5, color=t["ink_muted"], family="Menlo")
        if oact:
            b.icon("check", orow, size=16, color=t["brand_primary"])

    # 5. Right-Bottom: UserProfileMenu Popover (x=946, y=506, w=454, h=298)
    upm = b.frame(
        "Overlay/UserProfileMenu",
        artboard,
        946,
        506,
        454,
        298,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 14, "layout": "vertical", "mainAlign": "spaceBetween", "padding": [16, 18, 14, 18]},
        shadow=t["shadow_popover"],
    )
    uph = b.frame("UPH", upm, 0, 0, 418, 56, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 12})
    uav = b.frame("UAV", uph, 0, 0, 44, 44, {"fill": t["brand_primary"], "cornerRadius": 10, "layout": "horizontal", "mainAlign": "center", "crossAlign": "center"})
    b.text("AS", uav, size=16, color="#FFFFFF", style="Bold")
    utxt = b.frame("UTx", uph, 0, 0, 350, 44, {"fill": "none", "layout": "vertical", "gap": 3})
    b.text("Agga Sugitha", utxt, size=14, color=t["ink"], style="Bold")
    b.text("cfo@sinerginusantara.co.id · Super Admin / CFO", utxt, size=11.5, color=t["ink_muted"])

    perm_box = b.frame("RBACBox", upm, 0, 0, 418, 128, {"fill": t["surface_alt"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "gap": 8, "padding": [11, 13, 11, 13]})
    b.text("HAK AKSES & KONTEKS SESI AKTIF", perm_box, size=10, color=t["ink_muted"], style="Bold", letter_spacing="4%")
    pr1 = b.frame("PR1", perm_box, 0, 0, 392, 24, {"fill": "none", "layout": "horizontal", "gap": 6})
    for sc in ["gl.*", "reporting.read", "ap:approve", "currency:read"]:
        add_badge(b, pr1, sc, "info", t, dot=False, compact=True)
    b.text("Entitas Aktif: PT-SN · Sinergi Nusantara (Jakarta Pusat)", perm_box, size=11, color=t["ink_secondary"])
    b.text("Portal: Backoffice Operasional · Keamanan 2FA Aktif", perm_box, size=11, color=t["ink_muted"])

    up_foot = b.frame("UPFoot", upm, 0, 0, 418, 36, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    add_button(b, up_foot, "Preferensi & Keamanan", t, variant="outline", size="sm")
    add_button(b, up_foot, "Keluar Sesi", t, variant="danger-soft", size="sm")


# ==============================================================================
# 13. PAGE 2: DESIGN SYSTEM & UI KIT (2 Studio Boards)
# ==============================================================================


def build_design_system_page(b: DocBuilder, ds_page: str) -> None:
    b.page_switch(ds_page)
    t = LIGHT

    # --------------------------------------------------------------------------
    # Board 1 (x=0, y=0, 1480x1040): Design Tokens, Variables & Brand Identity
    # --------------------------------------------------------------------------
    b1 = b.frame(
        "01 · Design Tokens, Variables & Brand Identity",
        None,
        0,
        0,
        1480,
        1200,
        {"fill": t["canvas"], "layout": "vertical", "gap": 18, "padding": [28, 32, 32, 32], "clip": True},
    )

    # Top Title Banner
    b1_hdr = b.frame("DSHeader", b1, 0, 0, 1416, 60, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b1_hl = b.frame("HL", b1_hdr, 0, 0, 820, 56, {"fill": "none", "layout": "vertical", "gap": 4})
    b.text("Enterprise ERP Backoffice — Design System & Tokens (CONS-013)", b1_hl, size=22, color=t["ink"], style="Bold", letter_spacing="-2%")
    b.text("Ratified Stack: React 19 + Vite + TanStack (Router, Query v5, Virtual) + Zustand + Tailwind CSS v4 · Tri-Currency ADR-014", b1_hl, size=12, color=t["ink_muted"])
    b1_hr = b.frame("HR", b1_hdr, 0, 0, 380, 28, {"fill": "none", "layout": "horizontal", "mainAlign": "max", "crossAlign": "center", "gap": 8})
    add_badge(b, b1_hr, "Variables: Light & Dark", "brand", t, dot=True)
    add_badge(b, b1_hr, "7 Text Styles · 6 Paint Styles", "success", t, dot=True)

    # Row 1: Brand Lockup & Theme Mode Previews (h=228)
    r1 = b.frame("Row1/BrandAndModes", b1, 0, 0, 1416, 228, {"fill": "none", "layout": "horizontal", "gap": 18})

    brand_card = b.frame(
        "BrandIdentityCard",
        r1,
        0,
        0,
        520,
        228,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 18},
        shadow=t["shadow_card"],
    )
    b.text("Brand Mark & Asset Lockup", brand_card, size=14, color=t["ink"], style="Bold")
    chips = b.frame("BrandChips", brand_card, 0, 0, 484, 118, {"fill": "none", "layout": "horizontal", "gap": 12})

    # Real raster asset placed via image.place
    img_chip = b.frame("AssetLogoPNG", chips, 0, 0, 148, 118, {"fill": "#FFFFFF", "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "mainAlign": "center", "crossAlign": "center", "gap": 6})
    if BRAND_LOGO_PNG.exists():
        b.place_image(BRAND_LOGO_PNG, img_chip, width=68, height=68, mode="fit", name="brand-logo.png")
    b.text("public/brand-logo.png", img_chip, size=10, color=t["ink_muted"], family="Menlo")

    # Crisp Vector Prism on Light
    lt_chip = b.frame("VectorPrismLight", chips, 0, 0, 156, 118, {"fill": "#F8FAFC", "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "mainAlign": "center", "crossAlign": "center", "gap": 8})
    b.brand_prism_mark(lt_chip, size=48, dark_surface=False)
    b.text("Vector Prism · Light", lt_chip, size=10.5, color=t["ink_secondary"], style="SemiBold")

    # Crisp Vector Prism on Dark
    dk_chip = b.frame("VectorPrismDark", chips, 0, 0, 156, 118, {"fill": "#0F172A", "stroke": "#1E293B", "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "mainAlign": "center", "crossAlign": "center", "gap": 8})
    b.brand_prism_mark(dk_chip, size=48, dark_surface=True)
    b.text("Vector Prism · Dark", dk_chip, size=10.5, color="#CBD5E1", style="SemiBold")

    b.text("Vector mark keeps transparent background & razor edges at all DPI on Light & Dark navbars.", brand_card, size=11, color=t["ink_muted"])

    # Semantic Theme Live Variable Cards (Light Mode vs Dark Mode bound to Semantic Theme collection!)
    modes_card = b.frame(
        "ThemeModesMatrix",
        r1,
        0,
        0,
        878,
        228,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "gap": 12, "padding": 18},
        shadow=t["shadow_card"],
    )
    b.text("Semantic Theme Variable Modes (:root Light vs html.dark)", modes_card, size=14, color=t["ink"], style="Bold")
    dual = b.frame("DualModes", modes_card, 0, 0, 842, 160, {"fill": "none", "layout": "horizontal", "gap": 14})

    # Live Light Mode Container
    m_lt = b.frame("ModePreview/Light", dual, 0, 0, 414, 160, {"fill": LIGHT["canvas"], "stroke": LIGHT["border"], "strokeWeight": 1, "cornerRadius": 10, "padding": 14, "layout": "vertical", "mainAlign": "spaceBetween"}, bind_var=("fill", "bg/canvas"))
    b.set_mode(m_lt, "semantic", "Light")
    m_lt_top = b.frame("Top", m_lt, 0, 0, 386, 22, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Mode: Light (:root)", m_lt_top, size=12, color=LIGHT["ink"], style="Bold")
    b.text("--erp-canvas #F8FAFC", m_lt_top, size=10.5, color=LIGHT["ink_muted"], family="Menlo")
    m_lt_surf = b.frame("SurfCard", m_lt, 0, 0, 386, 98, {"fill": LIGHT["surface"], "stroke": LIGHT["border"], "strokeWeight": 1, "cornerRadius": 8, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 12}, bind_var=("fill", "bg/surface"))
    b.text("Saldo Buku Besar Terkonsolidasi", m_lt_surf, size=11.5, color=LIGHT["ink_muted"], style="SemiBold")
    b.text("Rp 48.250.400.000", m_lt_surf, size=18, color=LIGHT["ink"], style="Bold", tnum=True)
    b.text("border #E2E8F0 · card #FFFFFF · muted #F1F5F9", m_lt_surf, size=10, color=LIGHT["ink_muted"], family="Menlo")

    # Live Dark Mode Container
    m_dk = b.frame("ModePreview/Dark", dual, 0, 0, 414, 160, {"fill": DARK["canvas"], "stroke": DARK["border"], "strokeWeight": 1, "cornerRadius": 10, "padding": 14, "layout": "vertical", "mainAlign": "spaceBetween"}, bind_var=("fill", "bg/canvas"))
    b.set_mode(m_dk, "semantic", "Dark")
    m_dk_top = b.frame("Top", m_dk, 0, 0, 386, 22, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Mode: Dark (html.dark)", m_dk_top, size=12, color=DARK["ink"], style="Bold")
    b.text("--erp-canvas #020617", m_dk_top, size=10.5, color=DARK["ink_muted"], family="Menlo")
    m_dk_surf = b.frame("SurfCard", m_dk, 0, 0, 386, 98, {"fill": DARK["surface"], "stroke": DARK["border"], "strokeWeight": 1, "cornerRadius": 8, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 12}, bind_var=("fill", "bg/surface"))
    b.text("Saldo Buku Besar Terkonsolidasi", m_dk_surf, size=11.5, color=DARK["ink_muted"], style="SemiBold")
    b.text("Rp 48.250.400.000", m_dk_surf, size=18, color=DARK["ink"], style="Bold", tnum=True)
    b.text("hairline #1E293B · card #0F172A · muted #1E293B", m_dk_surf, size=10, color=DARK["ink_muted"], family="Menlo")

    # Row 2: Complete Color Scales (h=314)
    r2 = b.frame(
        "Row2/ColorScales",
        b1,
        0,
        0,
        1416,
        314,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "gap": 12, "padding": 18},
        shadow=t["shadow_card"],
    )
    b.text("Color System — Tailwind v4 @theme Tokens", r2, size=14, color=t["ink"], style="Bold")

    for scale_lbl, scale_dict in [("Brand Cobalt (brand-*)", BRAND_SCALE), ("Slate Neutral (slate-*)", SLATE_SCALE)]:
        sc_row = b.frame(f"Scale/{scale_lbl}", r2, 0, 0, 1380, 96, {"fill": "none", "layout": "vertical", "gap": 6})
        b.text(scale_lbl, sc_row, size=11.5, color=t["ink_secondary"], style="SemiBold")
        swatches = b.frame("Swatches", sc_row, 0, 0, 1380, 72, {"fill": "none", "layout": "horizontal", "gap": 8})
        for step, hx in scale_dict.items():
            sw = b.frame(f"Sw/{step}", swatches, 0, 0, 118, 72, {"fill": t["surface_alt"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 8, "layout": "vertical", "clip": True})
            b.rect("Chip", sw, 0, 0, 118, 40, {"fill": hx})
            meta = b.frame("M", sw, 0, 0, 118, 32, {"fill": "none", "layout": "vertical", "mainAlign": "center", "padding": [2, 8, 2, 8], "gap": 1})
            b.text(step, meta, size=10, color=t["ink"], style="Bold")
            b.text(hx, meta, size=9.5, color=t["ink_muted"], family="Menlo")

    # Semantic + Chart Swatch Row
    sem_row = b.frame("SemRow", r2, 0, 0, 1380, 56, {"fill": "none", "layout": "horizontal", "gap": 10})
    sem_chips = [
        ("Accent 500", "#F97316"),
        ("Accent 600", "#EA580C"),
        ("Success 600", "#059669"),
        ("Warning 600", "#D97706"),
        ("Danger 600", "#E11D48"),
        ("Chart Asset", "#0284C7"),
        ("Chart Equity", "#4338CA"),
        ("Chart Revenue", "#0F766E"),
    ]
    for slbl, shx in sem_chips:
        sc = b.frame(f"Sem/{slbl}", sem_row, 0, 0, 163.5, 52, {"fill": t["surface_alt"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 8, "layout": "horizontal", "crossAlign": "center", "gap": 10, "padding": [0, 10, 0, 10]})
        b.rect("Dot", sc, 0, 0, 26, 26, {"fill": shx, "cornerRadius": 6})
        stx = b.frame("Tx", sc, 0, 0, 108, 32, {"fill": "none", "layout": "vertical", "gap": 2})
        b.text(slbl, stx, size=10.5, color=t["ink"], style="SemiBold")
        b.text(shx, stx, size=9.5, color=t["ink_muted"], family="Menlo")

    # Row 3: Typography Scale & Elevation/Radius (h=428 — fits all 7 levels!)
    r3 = b.frame("Row3/TypeAndElevation", b1, 0, 0, 1416, 428, {"fill": "none", "layout": "horizontal", "gap": 18})

    # Typography Card (w=880)
    typo = b.frame(
        "TypographyScale",
        r3,
        0,
        0,
        880,
        428,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "gap": 9, "padding": 18},
        shadow=t["shadow_card"],
    )
    ty_hdr = b.frame("TYH", typo, 0, 0, 844, 24, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b.text("Typography Scale & Tabular Monetary Rule (.font-money)", ty_hdr, size=14, color=t["ink"], style="Bold")
    add_badge(b, ty_hdr, "Inter (tnum) + Menlo (tnum, zero)", "info", t, dot=False, compact=True)

    type_rows = [
        ("Display/H1", "Inter Bold · 21px · -2%", "Dashboard Eksekutif & Buku Besar", 21, "Inter", "Bold", False),
        ("KPI/Hero", "Inter Bold · 22px · tnum", "Rp 48.250.400.000", 22, "Inter", "Bold", True),
        ("Heading/Card", "Inter SemiBold · 14px", "Saldo Buku Besar per Kategori Akun", 14, "Inter", "SemiBold", False),
        ("Body/Default", "Inter Regular · 12.5px · 150%", "Pelunasan invoice pengadaan lisensi cloud Q4", 12.5, "Inter", "Regular", False),
        ("Label/Upper", "Inter Bold · 10px · +4%", "KEUANGAN & AKUNTANSI · TOTAL ASET GL", 10, "Inter", "Bold", False),
        ("Mono/Code", "Menlo Regular · 11.5px · zero", "JRN-2026-0842 · 2101-02 · USD @ 15.875", 11.5, "Menlo", "Regular", True),
        ("Mono/Money", "Menlo Bold · 12px · tnum zero", "190.500.000,00  |  -12.000,00 USD", 12, "Menlo", "Bold", True),
    ]
    for sname, spec, sample, fsz, ffam, fsty, is_num in type_rows:
        tr = b.frame(f"Type/{sname}", typo, 0, 0, 844, 42, {"fill": t["surface_alt"], "stroke": t["border_subtle"], "strokeWeight": 1, "cornerRadius": 7, "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 12, 0, 12]})
        tl = b.frame("TL", tr, 0, 0, 296, 22, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 10})
        b.text(sname, tl, size=11, color=t["brand_primary"], style="Bold", family="Menlo", width=116)
        b.text(spec, tl, size=10.5, color=t["ink_muted"], width=170)
        b.text(sample, tr, size=fsz, color=t["ink"], style=fsty, family=ffam, tnum=is_num, zero=(ffam == "Menlo"), truncate=True, width=510, align="right")

    # Elevation & Radius Card (w=518)
    elev = b.frame(
        "ElevationAndRadius",
        r3,
        0,
        0,
        518,
        428,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 18},
        shadow=t["shadow_card"],
    )
    b.text("Elevation Shadows & Corner Radii", elev, size=14, color=t["ink"], style="Bold")
    sh_row = b.frame("ShadowRow", elev, 0, 0, 482, 130, {"fill": t["canvas"], "stroke": t["border_subtle"], "strokeWeight": 1, "cornerRadius": 10, "layout": "horizontal", "mainAlign": "spaceEvenly", "crossAlign": "center", "padding": 14})
    for slbl, sfx in [("Card (y1 r3)", t["shadow_card"]), ("Popover (y12 r28)", t["shadow_popover"]), ("Modal 2XL (y22 r48)", t["shadow_modal"])]:
        sc = b.frame(f"Sh/{slbl}", sh_row, 0, 0, 138, 86, {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "mainAlign": "center", "crossAlign": "center", "gap": 4, "padding": 8}, shadow=sfx)
        b.text(slbl, sc, size=10.5, color=t["ink"], style="SemiBold", align="center")
        b.text("DropShadow", sc, size=9.5, color=t["ink_muted"], family="Menlo")

    rad_row = b.frame("RadiusRow", elev, 0, 0, 482, 118, {"fill": t["surface_alt"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 14})
    b.text("Border Radius & 4px Grid Rhythm", rad_row, size=12, color=t["ink"], style="SemiBold")
    rchips = b.frame("RChips", rad_row, 0, 0, 454, 62, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    for rlbl, rad in [("r=6px\nBadge", 6), ("r=8px\nInput/Btn", 8), ("r=12px\nCard", 12), ("r=16px\nDialog", 16), ("r=99px\nPill", 30)]:
        rc = b.frame(f"Rad/{rad}", rchips, 0, 0, 82, 58, {"fill": t["brand_soft"], "stroke": t["brand_soft_border"], "strokeWeight": 1.2, "cornerRadius": rad, "layout": "vertical", "mainAlign": "center", "crossAlign": "center"})
        b.text(rlbl, rc, size=10, color=t["brand_soft_ink"], style="SemiBold", align="center")

    # --------------------------------------------------------------------------
    # Board 2 (x=1620, y=0, 1480x1040): UI Component Library & Interactive States
    # --------------------------------------------------------------------------
    b2 = b.frame(
        "02 · UI Component Library & Interactive States",
        None,
        1620,
        0,
        1480,
        1200,
        {"fill": t["canvas"], "layout": "vertical", "gap": 18, "padding": [28, 32, 32, 32], "clip": True},
    )

    b2_hdr = b.frame("CompHeader", b2, 0, 0, 1416, 56, {"fill": "none", "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center"})
    b2_hl = b.frame("HL", b2_hdr, 0, 0, 820, 52, {"fill": "none", "layout": "vertical", "gap": 3})
    b.text("UI Component Library & Interactive States", b2_hl, size=22, color=t["ink"], style="Bold", letter_spacing="-2%")
    b.text("Component Sets (StatusBadge Variants) · Button States · Input / MoneyInput States · KpiCard · Toaster · 24 Vector Icons", b2_hl, size=12, color=t["ink_muted"])
    add_badge(b, b2_hdr, "FigCraft Component Sets", "brand", t, dot=True)

    # Row 1: Button Matrix (Left 820) + StatusBadge Component Set & Toasts (Right 578)
    c_r1 = b.frame("CompRow1", b2, 0, 0, 1416, 404, {"fill": "none", "layout": "horizontal", "gap": 18})

    btn_card = b.frame(
        "Comp/ButtonMatrix",
        c_r1,
        0,
        0,
        820,
        404,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "gap": 10, "padding": 18},
        shadow=t["shadow_card"],
    )
    b.text("Button.tsx — Variants × Interactive States × Sizes", btn_card, size=14, color=t["ink"], style="Bold")
    st_hdr = b.frame("StHdr", btn_card, 0, 0, 784, 18, {"fill": "none", "layout": "horizontal", "gap": 12})
    for col_l, col_w in [("VARIANT", 90), ("DEFAULT", 126), ("HOVER", 126), ("FOCUS RING", 126), ("DISABLED", 126), ("LOADING", 130)]:
        cf = b.frame("CH", st_hdr, 0, 0, col_w, 18, {"fill": "none", "layout": "horizontal", "crossAlign": "center"})
        b.text(col_l, cf, size=9.5, color=t["ink_muted"], style="Bold", letter_spacing="4%")

    for vname, vkey in [("Primary", "primary"), ("Secondary", "secondary"), ("Outline", "outline"), ("Ghost", "ghost"), ("Danger", "danger"), ("Success", "success")]:
        brow = b.frame(f"BRow/{vname}", btn_card, 0, 0, 784, 34, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 12, "clip": False})
        bl = b.frame("VL", brow, 0, 0, 90, 32, {"fill": "none", "layout": "horizontal", "crossAlign": "center"})
        b.text(vname, bl, size=11.5, color=t["ink_secondary"], style="SemiBold")
        for st in ["default", "hover", "focus", "disabled"]:
            w_slot = b.frame("Slot", brow, 0, 0, 126, 34, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "clip": False})
            add_button(b, w_slot, "Posting", t, variant=vkey, size="sm", icon_name="check" if vkey in ("primary", "success") else None, state=st)
        w_load = b.frame("SlotLoad", brow, 0, 0, 130, 34, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "clip": False})
        add_button(b, w_load, "Menyimpan", t, variant=vkey, size="sm", state="loading")

    # Size Strip
    sz_strip = b.frame("SizeStrip", btn_card, 0, 0, 784, 54, {"fill": t["surface_alt"], "stroke": t["border_subtle"], "strokeWeight": 1, "cornerRadius": 8, "layout": "horizontal", "crossAlign": "center", "gap": 14, "padding": [0, 14, 0, 14]})
    b.text("SIZES:", sz_strip, size=10, color=t["ink_muted"], style="Bold")
    for sz in ["xs", "sm", "md", "lg"]:
        add_button(b, sz_strip, f"Size {sz.upper()}", t, variant="primary", size=sz)

    # Right 578: StatusBadge Component Set + Toaster Alerts
    right_top = b.frame("Comp/BadgesAndToasts", c_r1, 0, 0, 578, 404, {"fill": "none", "layout": "vertical", "gap": 14})

    badge_card = b.frame(
        "StatusBadgeCard",
        right_top,
        0,
        0,
        578,
        152,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 16},
        shadow=t["shadow_card"],
    )
    b.text("StatusBadge.tsx — FigCraft Component Set (6 Variant Tones)", badge_card, size=13.5, color=t["ink"], style="Bold")
    badge_wrap = b.frame("BadgeVariants", badge_card, 0, 0, 546, 52, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8, "padding": [8, 12, 8, 12]})
    badge_comps: list[str] = []
    for t_lbl, t_key in [("Draf", "Neutral"), ("Posted", "Success"), ("Menunggu", "Warning"), ("Dibalik", "Danger"), ("Terbuka", "Info"), ("Balanced", "Brand")]:
        bg_frame = add_badge(b, badge_wrap, t_lbl, t_key.lower(), t, dot=True, name=f"Tone={t_key}")
        comp_alias = b.create_component(bg_frame)
        badge_comps.append(comp_alias)
    b.combine_variants(badge_comps, "StatusBadgeSet")
    b.text("Component Set `StatusBadgeSet` exported with 6 semantic variants & live 6px status indicator dot.", badge_card, size=11, color=t["ink_muted"])

    toast_card = b.frame(
        "ToasterCard",
        right_top,
        0,
        0,
        578,
        238,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 16},
        shadow=t["shadow_card"],
    )
    b.text("Toaster / Alert Notifications", toast_card, size=13.5, color=t["ink"], style="Bold")
    for t_tone, t_ico, t_msg in [
        ("success", "check-circle-2", "JRN-2026-0842 berhasil diposting ke Buku Besar."),
        ("warning", "alert-triangle", "Kurs BI USD/IDR belum diperbarui untuk hari libur bursa."),
        ("danger", "alert-circle", "Gagal posting: Total Debit & Kredit selisih Rp 50.000."),
    ]:
        tbg, tbdr, _, tfg = t["badges"][t_tone]
        tf = b.frame(f"Toast/{t_tone}", toast_card, 0, 0, 546, 38, {"fill": tbg, "stroke": tbdr, "strokeWeight": 1, "cornerRadius": 8, "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 12, 0, 12]})
        tl = b.frame("L", tf, 0, 0, 480, 22, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8})
        b.icon(t_ico, tl, size=14.5, color=tfg)
        b.text(t_msg, tl, size=11.5, color=tfg, style="SemiBold", truncate=True, width=445)
        b.icon("x", tf, size=13, color=tfg)

    # Row 2: Inputs & MoneyInput States + Sidebar NavItem States (h=274)
    c_r2 = b.frame("CompRow2", b2, 0, 0, 1416, 274, {"fill": "none", "layout": "horizontal", "gap": 18})

    inp_card = b.frame(
        "Comp/InputsAndMoney",
        c_r2,
        0,
        0,
        926,
        274,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "gap": 14, "padding": 18},
        shadow=t["shadow_card"],
    )
    b.text("Input.tsx & MoneyInput.tsx — Interactive Form States", inp_card, size=14, color=t["ink"], style="Bold")
    ir1 = b.frame("IR1", inp_card, 0, 0, 890, 80, {"fill": "none", "layout": "horizontal", "gap": 16})
    add_input_field(b, ir1, "Default Input", "INV-VND-2026-1192", t, width=286, input_h=36, mono=True)
    add_input_field(b, ir1, "Focus State (ring-brand-600)", "PT Sinergi Nusantara", t, width=286, input_h=36, state="focus")
    add_input_field(b, ir1, "Error State (Validasi)", "12.000,5", t, width=286, input_h=36, state="error", helper="Nominal melebihi batas pagu otorisasi", mono=True)

    ir2 = b.frame("IR2", inp_card, 0, 0, 890, 80, {"fill": "none", "layout": "horizontal", "gap": 16})
    add_input_field(b, ir2, "Search Input", "Cari nomor jurnal, memo...", t, width=286, input_h=36, left_icon="search", state="placeholder")
    add_input_field(b, ir2, "MoneyInput (IDR Prefix)", "190.500.000,00", t, width=286, input_h=36, currency_chip="IDR", mono=True, right_align=True)
    add_input_field(b, ir2, "MoneyInput (USD Focus)", "12.000,00", t, width=286, input_h=36, currency_chip="USD", state="focus", mono=True, right_align=True)

    # Sidebar NavItem States + MobileBottomNav Strip (w=472)
    nav_card = b.frame(
        "Comp/NavStates",
        c_r2,
        0,
        0,
        472,
        274,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 18},
        shadow=t["shadow_card"],
    )
    b.text("Sidebar NavItem & MobileBottomNav", nav_card, size=14, color=t["ink"], style="Bold")
    nv_list = b.frame("NVList", nav_card, 0, 0, 436, 126, {"fill": t["surface_alt"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "vertical", "gap": 5, "padding": 10})
    for nlbl, nico, nst, ncnt in [
        ("Dashboard Eksekutif (Active)", "layout-dashboard", "active", None),
        ("Hutang Usaha (AP) (Default + Badge)", "arrow-down-left", "default", "6"),
        ("Mata Uang & Kurs FX (Hover)", "coins", "hover", None),
    ]:
        nbg = t["brand_soft"] if nst == "active" else (t["surface"] if nst == "hover" else "none")
        nbdr = t["brand_soft_border"] if nst == "active" else (t["border"] if nst == "hover" else "none")
        nr = b.frame(f"NVS/{nlbl}", nv_list, 0, 0, 416, 32, {"fill": nbg, "stroke": nbdr, "strokeWeight": 1 if nbdr != "none" else 0, "cornerRadius": 7, "layout": "horizontal", "mainAlign": "spaceBetween", "crossAlign": "center", "padding": [0, 10, 0, 8]})
        nl = b.frame("L", nr, 0, 0, 340, 22, {"fill": "none", "layout": "horizontal", "crossAlign": "center", "gap": 8})
        b.rect("Ind", nl, 0, 0, 3, 16, {"fill": t["brand_primary"] if nst == "active" else "none", "cornerRadius": 2})
        b.icon(nico, nl, size=14.5, color=t["brand_primary"] if nst == "active" else t["ink_secondary"])
        b.text(nlbl, nl, size=12, color=t["brand_soft_ink"] if nst == "active" else t["ink"], style="SemiBold" if nst == "active" else "Regular")
        if ncnt:
            add_badge(b, nr, ncnt, "neutral", t, dot=False, compact=True)

    # Mini MobileBottomNav specimen with explicit section label
    b.text("MOBILE BOTTOM NAVIGATION SPECIMEN (< 768PX)", nav_card, size=9.5, color=t["ink_muted"], style="Bold", letter_spacing="4%")
    mb_spec = b.frame("MobileNavSpec", nav_card, 0, 0, 436, 52, {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 10, "layout": "horizontal", "mainAlign": "spaceEvenly", "crossAlign": "center"})
    for mlbl, mico, mact in [("Dashboard", "layout-dashboard", True), ("Jurnal", "book-open", False), ("AP/AR", "receipt", False), ("Otorisasi", "shield-check", False), ("Menu", "menu", False)]:
        mi = b.frame("MI", mb_spec, 0, 0, 72, 42, {"fill": "none", "layout": "vertical", "mainAlign": "center", "crossAlign": "center", "gap": 2})
        b.icon(mico, mi, size=15, color=t["brand_primary"] if mact else t["ink_muted"])
        b.text(mlbl, mi, size=10, color=t["brand_primary"] if mact else t["ink_muted"], style="Bold" if mact else "Regular")

    # Row 3: 4 KPI Card Specimens + Complete 24-Icon Lucide Vector Inventory (h=338)
    c_r3 = b.frame("CompRow3", b2, 0, 0, 1416, 338, {"fill": "none", "layout": "vertical", "gap": 14})

    kpi_comp_row = b.frame("KpiSpecimens", c_r3, 0, 0, 1416, 112, {"fill": "none", "layout": "horizontal", "gap": 16})
    add_kpi_card(b, kpi_comp_row, "VARIANT: BRAND", "Rp 48,25 M", "Aset Buku Besar", "+6,4%", "success", "brand", "landmark", t, width=342, height=112)
    add_kpi_card(b, kpi_comp_row, "VARIANT: WARNING", "Rp 14,82 M", "Hutang Dagang AP", "6 Tempo", "warning", "warning", "arrow-down-left", t, width=342, height=112)
    add_kpi_card(b, kpi_comp_row, "VARIANT: SUCCESS", "Rp 29,40 M", "Pendapatan AR", "+14,2%", "success", "success", "trending-up", t, width=342, height=112)
    add_kpi_card(b, kpi_comp_row, "VARIANT: SKY", "Rp 12,18 M", "Kas & Valas", "4 Rekening", "brand", "sky", "coins", t, width=342, height=112)

    # Vector Icon Inventory Sheet
    ic_card = b.frame(
        "Comp/LucideVectorSheet",
        c_r3,
        0,
        0,
        1416,
        212,
        {"fill": t["surface"], "stroke": t["border"], "strokeWeight": 1, "cornerRadius": 12, "layout": "vertical", "mainAlign": "spaceBetween", "padding": 16},
        shadow=t["shadow_card"],
    )
    b.text("Lucide Vector Iconography Sheet (Scaled 16×16 Vector Paths · 1.6px Stroke)", ic_card, size=13.5, color=t["ink"], style="Bold")
    icon_keys = list(LUCIDE_24.keys())[:24]
    for row_icons in [icon_keys[:12], icon_keys[12:24]]:
        irow = b.frame("IconRow", ic_card, 0, 0, 1384, 58, {"fill": "none", "layout": "horizontal", "gap": 8})
        for ik in row_icons:
            cell = b.frame(f"IcCell/{ik}", irow, 0, 0, 108, 58, {"fill": t["surface_alt"], "stroke": t["border_subtle"], "strokeWeight": 1, "cornerRadius": 8, "layout": "vertical", "mainAlign": "center", "crossAlign": "center", "gap": 5})
            b.icon(ik, cell, size=16, color=t["ink"], stroke_weight=1.6)
            b.text(ik, cell, size=9, color=t["ink_muted"], family="Menlo", truncate=True, width=100, align="center")


# ==============================================================================
# 14. DOCUMENT ASSEMBLY & HEADLESS CLI RUNNER
# ==============================================================================


def build_document_commands() -> list[dict[str, Any]]:
    b = DocBuilder()
    b.page_rename_current("01 Frontend UI (Backoffice SPA)")
    register_tokens_and_styles(b)

    # 3 x 2 Studio Grid of 1440x940 Artboards on Page 1
    ab1 = b.frame("01 · Dashboard Eksekutif (Light)", None, 0, 0, 1440, 940, {"fill": LIGHT["canvas"], "clip": True}, bind_var=("fill", "bg/canvas"))
    build_dashboard_artboard(b, ab1, LIGHT)

    ab2 = b.frame("02 · Buku Besar & Jurnal — Daftar & Filter", None, 1560, 0, 1440, 940, {"fill": LIGHT["canvas"], "clip": True}, bind_var=("fill", "bg/canvas"))
    build_journals_workspace(b, ab2, LIGHT, selected_idx=0)

    ab3 = b.frame("03 · SideDrawer (XL) — Detail Jurnal", None, 3120, 0, 1440, 940, {"fill": LIGHT["canvas"], "clip": True}, bind_var=("fill", "bg/canvas"))
    build_drawer_artboard(b, ab3, LIGHT)

    ab4 = b.frame("04 · AppDialog (2XL) — Buat Entri Jurnal", None, 0, 1100, 1440, 940, {"fill": LIGHT["canvas"], "clip": True}, bind_var=("fill", "bg/canvas"))
    build_dialog_artboard(b, ab4, LIGHT)

    ab5 = b.frame("05 · Dashboard Eksekutif (Dark Mode · html.dark)", None, 1560, 1100, 1440, 940, {"fill": DARK["canvas"], "clip": True}, bind_var=("fill", "bg/canvas"))
    b.set_mode(ab5, "semantic", "Dark")
    build_dashboard_artboard(b, ab5, DARK)

    ab6 = b.frame("06 · Responsif Mobile (390px) & Overlay Interaktif", None, 3120, 1100, 1440, 940, {"fill": "#EEF2F7", "clip": True})
    build_mobile_and_overlays_artboard(b, ab6, LIGHT)

    # Page 2: Design System & UI Kit
    pg2 = b.page_create("02 Design System & UI Kit")
    build_design_system_page(b, pg2)

    return b.cmds


def main() -> None:
    parser = argparse.ArgumentParser(description="Build ERP Backoffice SPA FigCraft design.")
    parser.add_argument(
        "--headless",
        nargs="?",
        const=str(DEFAULT_OUT),
        default=str(DEFAULT_OUT),
        help="Output .figcraft file path",
    )
    args = parser.parse_args()
    out_path = Path(args.headless).resolve()
    out_path.parent.mkdir(parents=True, exist_ok=True)

    cmds = build_document_commands()
    with tempfile.NamedTemporaryFile("w", suffix=".json", dir=str(out_path.parent), delete=False) as tf:
        json.dump(cmds, tf)
        script_path = Path(tf.name)

    try:
        res = subprocess.run(
            ["figcraft-cli", "run", "--out", str(out_path), str(script_path)],
            capture_output=True,
            text=True,
            check=False,
        )
        if res.returncode != 0:
            print(res.stdout)
            print(res.stderr)
            raise SystemExit(res.returncode)
        lines = [ln for ln in res.stdout.splitlines() if ln.strip()]
        print(f"Executed {len(cmds)} commands -> wrote {out_path} ({len(lines)} output lines)")
    finally:
        script_path.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
