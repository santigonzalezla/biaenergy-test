import styles from './login.module.css';
import {useEffect} from 'react';
import {Moon, Sparkles, Sun, Zap} from 'lucide-react';
import LoginForm from '@/components/login/loginform/LoginForm.tsx';
import {useTheme} from '@/context/theme.ts';
import {APP_NAME} from '@/lib/constants.ts';

const PIPELINE = ['Datos', 'Análisis', 'Anomalía', 'Explicación', 'Prioridad', 'Acción'];

const CAPABILITIES = [
    'Detecta cambios de consumo contra una línea base robusta por hora.',
    'Distingue anomalías reales de cortes programados y cambios operativos.',
    'Explica cada hallazgo con evidencia eléctrica y recomienda una acción.',
];

const LoginPage = () =>
{
    const {theme, toggleTheme} = useTheme();
    const isDark = theme === 'dark';

    useEffect(() =>
    {
        document.title = `Ingresar · ${APP_NAME}`;
    }, []);

    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <section className={styles.brandPanel}>
                    <div className={styles.glowTop}/>
                    <div className={styles.glowBottom}/>

                    <div className={styles.brand}>
                        <div className={styles.logo}>
                            <Zap size={20} strokeWidth={2.5}/>
                        </div>
                        <span className={styles.wordmark}>bia <span>energy</span></span>
                    </div>

                    <div className={styles.pitch}>
                        <span className={styles.badge}>
                            <Sparkles size={12}/>
                            Anomaly Center
                        </span>
                        <h1 className={styles.headline}>Detecta, explica y prioriza anomalías de consumo con IA.</h1>
                        <ul className={styles.capabilities}>
                            {CAPABILITIES.map(capability => <li key={capability}>{capability}</li>)}
                        </ul>
                    </div>

                    <ol className={styles.pipeline} aria-label='Pipeline de análisis'>
                        {PIPELINE.map(step => <li key={step}>{step}</li>)}
                    </ol>
                </section>

                <section className={styles.formPanel}>
                    <button
                        type='button'
                        className={styles.themeToggle}
                        onClick={toggleTheme}
                        aria-label={isDark ? 'Cambiar a modo claro' : 'Cambiar a modo oscuro'}
                    >
                        {isDark ? <Sun size={16}/> : <Moon size={16}/>}
                    </button>

                    <div className={styles.formContent}>
                        <header className={styles.formHeader}>
                            <h2 className={styles.formTitle}>Ingresar</h2>
                            <p className={styles.formDescription}>Accede con tu cuenta para revisar el estado de los medidores.</p>
                        </header>
                        <LoginForm/>
                    </div>
                </section>
            </div>
        </div>
    );
}

export default LoginPage;
