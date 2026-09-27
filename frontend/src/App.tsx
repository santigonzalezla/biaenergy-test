import {Toaster} from 'sonner';
import AuthProvider from '@/context/AuthContext.tsx';
import ThemeProvider from '@/context/ThemeContext.tsx';
import {useTheme} from '@/context/theme.ts';
import AppRouter from '@/router/AppRouter.tsx';

const ThemedToaster = () =>
{
    const {theme} = useTheme();

    return <Toaster theme={theme} position='bottom-right' richColors closeButton/>;
}

const App = () =>
{
    return (
        <ThemeProvider>
            <AuthProvider>
                <AppRouter/>
                <ThemedToaster/>
            </AuthProvider>
        </ThemeProvider>
    );
}

export default App;
