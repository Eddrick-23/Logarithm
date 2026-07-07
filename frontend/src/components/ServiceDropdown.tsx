import { FormControl, InputLabel, MenuItem, Select, type SelectChangeEvent } from "@mui/material";
import { memo } from "react";
import DropdownItemContent from "./DropdownItemContent";

interface ServiceDropdownProps {
    services: string[];
    onChange: (event: SelectChangeEvent<string[]>) => void;
    isLoading: boolean;
    serviceOptions: string[];
}

export default memo(function ServiceDropdown({ services, onChange, isLoading, serviceOptions }: ServiceDropdownProps) {
    return (
        <FormControl variant="outlined" sx={{ width: 200 }}>
            <InputLabel shrink>Services</InputLabel>
            {/* value in InputLabel must match label in Select */}
            <Select
                label="Services"
                multiple
                displayEmpty
                value={services}
                size="small"
                onChange={onChange}
                disabled={isLoading}
                aria-label="services"
                renderValue={(selected) =>
                    selected.length === 0 ? (isLoading ? "Loading..." : "All services") : selected.join(", ")
                }
            >
                {serviceOptions.length === 0 ? (
                    <MenuItem disabled>No services found</MenuItem>
                ) : (
                    serviceOptions.map((serviceOption) => (
                        <MenuItem key={serviceOption} value={serviceOption}>
                            <DropdownItemContent option={serviceOption} checked={services.includes(serviceOption)} />
                        </MenuItem>
                    ))
                )}
            </Select>
        </FormControl>
    );
});
