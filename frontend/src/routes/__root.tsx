import { Outlet, createRootRoute } from "@tanstack/react-router";
import Box from "@mui/material/Box";
import MiniDrawer from "../components/MiniDrawer";
import Toolbar from "@mui/material/Toolbar"; // ← add this

export const Route = createRootRoute({
    component: () => (
        <Box sx={{ display: "flex" }}>
            <MiniDrawer />
            <Box
                component="main"
                sx={{
                    flexGrow: 1,
                    p: 3,
                    minWidth: 0,
                }}
            >
                <Toolbar />
                <Outlet />
            </Box>
        </Box>
    ),
});
