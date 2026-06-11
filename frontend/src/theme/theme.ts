import { createTheme } from "@mui/material";

export const theme = createTheme({
    palette: {
        mode: "dark",

        background: {
            default: "#1e293b",
            paper: "#334155",
        },

        divider: "#475569",

        primary: {
            main: "#38bdf8",
            contrastText: "#0f172a",
        },

        text: {
            primary: "#f8fafc",
            secondary: "#cbd5e1",
            disabled: "#94a3b8",
        },

        success: {
            main: "#34d399",
        },

        warning: {
            main: "#fbbf24",
        },

        error: {
            main: "#f87171",
        },
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
        // TODO: override the colour scheme of the entire page so we dont need to do this
        MuiAlert: {
            styleOverrides: {
                root: ({ ownerState }) => ({
                    ...(ownerState.severity === "error" && {
                        backgroundColor: "#fef2f2",
                        color: "#5f2120",
                        "& .MuiAlert-icon": { color: "#ef5350" },
                    }),
                    ...(ownerState.severity === "warning" && {
                        backgroundColor: "#fff4e5",
                        color: "#663c00",
                        "& .MuiAlert-icon": { color: "#ff9800" },
                    }),
                    ...(ownerState.severity === "info" && {
                        backgroundColor: "#e5f6fd",
                        color: "#014361",
                        "& .MuiAlert-icon": { color: "#0288d1" },
                    }),
                    ...(ownerState.severity === "success" && {
                        backgroundColor: "#edf7ed",
                        color: "#1e4620",
                        "& .MuiAlert-icon": { color: "#4caf50" },
                    }),
                }),
            },
        },
    },
});
