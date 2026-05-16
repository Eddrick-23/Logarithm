import { createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";
import { ThemeProvider } from "@mui/material/styles";
import { CssBaseline } from "@mui/material";
import { theme } from "./theme/theme";
import { RouterProvider } from "@tanstack/react-router";

const router = createRouter({ routeTree });

function App() {
    return (
        <ThemeProvider theme={theme}>
            <RouterProvider router={router} />
            <CssBaseline />
        </ThemeProvider>
    );
}

export default App;
