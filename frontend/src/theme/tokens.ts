import type { SxProps, Theme } from "@mui/material/styles";

export const card: SxProps<Theme> = {
    bgcolor: "background.paper",
    border: "1px solid",
    borderColor: "divider",
    borderRadius: "10px",
    p: "14px 16px",
};

export const sectionLabel: SxProps<Theme> = {
    fontSize: 14,
    fontWeight: 700,
    textTransform: "uppercase",
    letterSpacing: "0.1em",
    color: "text.disabled",
    mb: 1,
};

export const statValue: SxProps<Theme> = {
    fontSize: 24,
    fontWeight: 700,
    color: "text.primary",
    lineHeight: 1.1,
    my: 0.5,
};

// Grid layout for a single log row (time | service | type | message)
export const logRowSx: SxProps<Theme> = {
    display: "grid",
    gridTemplateColumns: "100px 110px 64px 1fr",
    gap: "8px",
    alignItems: "baseline",
    py: 0.6,
    borderBottom: "1px solid rgba(255,255,255,0.04)",
    "&:last-child": { borderBottom: "none" },
};

// Pulse keyframe shared by the live-indicator dot
export const pulseSx: SxProps<Theme> = {
    width: 7,
    height: 7,
    borderRadius: "50%",
    bgcolor: "success.main",
    boxShadow: "0 0 5px currentColor",
    animation: "sdPulse 1.8s infinite",
    "@keyframes sdPulse": {
        "0%, 100%": { opacity: 1 },
        "50%": { opacity: 0.3 },
    },
};
