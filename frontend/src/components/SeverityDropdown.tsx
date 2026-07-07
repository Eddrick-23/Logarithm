import { FormControl, InputLabel, MenuItem, Select, type SelectChangeEvent } from "@mui/material";
import type { LogType } from "../types/Log";
import { memo } from "react";
import DropdownItemContent from "./DropdownItemContent";

interface SeverityDropdownProps {
    severities: LogType[];
    onChange: (event: SelectChangeEvent<LogType[]>) => void;
}

const LOG_TYPES: LogType[] = ["trace", "debug", "info", "warn", "error", "fatal"];

export default memo(function SeverityDropdown({ severities, onChange }: SeverityDropdownProps) {
    return (
        <FormControl variant="outlined" sx={{ width: 160 }}>
            <InputLabel shrink>Severities</InputLabel>
            {/* value in InputLabel must match label in Select */}
            <Select
                label="Severities"
                multiple
                displayEmpty
                value={severities}
                size="small"
                onChange={onChange}
                aria-label="severity level"
                renderValue={(selected) =>
                    selected.length === 0 ? "All severities" : selected.map((s) => s.toUpperCase()).join(", ")
                }
            >
                {LOG_TYPES.map((type) => (
                    <MenuItem key={type} value={type}>
                        <DropdownItemContent option={type.toUpperCase()} checked={severities.includes(type)} />
                    </MenuItem>
                ))}
            </Select>
        </FormControl>
    );
});
