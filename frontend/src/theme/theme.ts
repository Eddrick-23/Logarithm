import { createTheme } from "@mui/material";

export const theme = createTheme({
    palette: {
        mode: "dark",

        background: {
            default: "#121212",
            paper: "#1e1e1e",
        },

        divider: "#30363d",

        primary: {
            main: "#2196F3",
            contrastText: "#ffffff",
        },

        text: {
            primary: "#e1e1e1",
            secondary: "#8b949e",
            disabled: "#9e9e9e",
        },

        success: { main: "#22c55e" },
        warning: { main: "#eab308" },
        error: { main: "#ef4444" },
    },

    typography: {
        fontFamily: "'Inter', sans-serif",
    },

    shape: { borderRadius: 10 },

    components: {
        // Remove gradient overlay in MUI Paper
        MuiPaper: {
            styleOverrides: {
                root: { backgroundImage: "none" },
            },
        },

        MuiCard: {
            styleOverrides: {
                root: ({ theme }) => ({
                    background: theme.palette.background.paper,
                    border: `1px solid ${theme.palette.divider}`,
                    borderRadius: theme.shape.borderRadius,
                }),
            },
        },

        // Dividers use the same surface-lighter colour
        MuiDivider: {
            styleOverrides: {
                root: ({ theme }) => ({ borderColor: theme.palette.divider }),
            },
        },

        // Typography: section labels are always the muted/disabled colour
        MuiTypography: {
            defaultProps: { variantMapping: { body1: "p", body2: "p" } },
        },

        // Alert: override it with light mode colours
        MuiAlert: {
            styleOverrides: {
                root: ({ ownerState }) => ({
                    borderWidth: "1px",
                    borderStyle: "solid",
                    ...(ownerState.severity === "error" && {
                        backgroundColor: "rgba(239,68,68,0.14)",
                        borderColor: "rgba(239,68,68,0.32)",
                        color: "#fca5a5",
                        "& .MuiAlert-icon": { color: "#ef4444" },
                    }),
                    ...(ownerState.severity === "warning" && {
                        backgroundColor: "rgba(234,179,8,0.12)",
                        borderColor: "rgba(234,179,8,0.28)",
                        color: "#fde047",
                        "& .MuiAlert-icon": { color: "#eab308" },
                    }),
                    ...(ownerState.severity === "info" && {
                        backgroundColor: "rgba(14,165,233,0.11)",
                        borderColor: "rgba(14,165,233,0.27)",
                        color: "#7dd3fc",
                        "& .MuiAlert-icon": { color: "#0ea5e9" },
                    }),
                    ...(ownerState.severity === "success" && {
                        backgroundColor: "rgba(34,197,94,0.11)",
                        borderColor: "rgba(34,197,94,0.27)",
                        color: "#86efac",
                        "& .MuiAlert-icon": { color: "#22c55e" },
                    }),
                }),
            },
        },
    },
});
