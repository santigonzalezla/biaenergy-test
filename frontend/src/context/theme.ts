import {createContext, useContext} from 'react';

export type Theme = 'light' | 'dark';

export const THEME_STORAGE_KEY = 'biaenergy.theme';

export interface ThemeContextType {
    theme: Theme;
    toggleTheme: () => void;
    setTheme: (theme: Theme) => void;
}

export const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export const useTheme = () =>
{
    const context = useContext(ThemeContext);

    if (context === undefined) throw new Error('useTheme must be used within a ThemeProvider');

    return context;
}
