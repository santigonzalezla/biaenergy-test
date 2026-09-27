import {useCallback, useEffect, useMemo, useState, type ReactNode} from 'react';
import {ThemeContext, THEME_STORAGE_KEY, type Theme} from '@/context/theme.ts';

interface ThemeProviderProps {
    children: ReactNode;
}

const readInitialTheme = (): Theme =>
{
    return document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'light';
}

const ThemeProvider = ({children}: ThemeProviderProps) =>
{
    const [theme, setTheme] = useState<Theme>(readInitialTheme);

    useEffect(() =>
    {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem(THEME_STORAGE_KEY, theme);
    }, [theme]);

    const toggleTheme = useCallback(() => setTheme(previous => previous === 'light' ? 'dark' : 'light'), []);

    const value = useMemo(() => ({theme, toggleTheme, setTheme}), [theme, toggleTheme]);

    return (
        <ThemeContext.Provider value={value}>
            {children}
        </ThemeContext.Provider>
    );
}

export default ThemeProvider;
