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
        // Inject Google Fonts
        MuiCssBaseline: {
            styleOverrides: `
                @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;700&display=swap');
            `,
        },

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
    },
});
