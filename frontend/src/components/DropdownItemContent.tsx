import { Checkbox, ListItemText } from "@mui/material";
import { memo } from "react";

interface DropdownItemContentProps {
    option: string;
    checked: boolean;
}

export default memo(function DropdownItemContent({ option, checked }: DropdownItemContentProps) {
    return (
        <>
            <Checkbox checked={checked} size="small" />
            <ListItemText primary={option} />
        </>
    );
});
