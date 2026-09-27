import styles from './loginform.module.css';
import {useState, type FormEvent} from 'react';
import {AlertCircle, Clock, Eye, EyeOff, Loader2, Lock, LogIn, Mail, X} from 'lucide-react';
import {useAuth} from '@/context/auth.ts';
import {ApiError} from '@/lib/api.ts';

type FieldErrors = Partial<Record<'email' | 'password', string>>;

const LoginForm = () =>
{
    const {login, sessionExpired, dismissExpiredNotice} = useAuth();
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

    const handleSubmit = async (event: FormEvent<HTMLFormElement>) =>
    {
        event.preventDefault();

        if (isSubmitting) return;

        setIsSubmitting(true);
        setError(null);
        setFieldErrors({});
        dismissExpiredNotice();

        try
        {
            await login({email, password});
        }
        catch (caught)
        {
            const apiError = ApiError.from(caught);

            setError(apiError.message);

            if (apiError.code === 'VALIDATION_ERROR' && apiError.details && !Array.isArray(apiError.details))
            {
                setFieldErrors({email: apiError.details.email, password: apiError.details.password});
            }

            setIsSubmitting(false);
        }
    }

    return (
        <form className={styles.form} onSubmit={handleSubmit} noValidate>
            {sessionExpired && (
                <div className={`${styles.notice} ${styles.info}`} role='status'>
                    <Clock size={16}/>
                    <p>Tu sesión expiró. Vuelve a ingresar para continuar.</p>
                </div>
            )}

            {error && (
                <div className={`${styles.notice} ${styles.error}`} role='alert'>
                    <AlertCircle size={16}/>
                    <p>{error}</p>
                    <button type='button' className={styles.dismiss} onClick={() => setError(null)} aria-label='Cerrar mensaje'>
                        <X size={14}/>
                    </button>
                </div>
            )}

            <div className={styles.field}>
                <label htmlFor='email' className={styles.label}>Correo electrónico</label>
                <div className={`${styles.inputWrapper} ${fieldErrors.email ? styles.invalid : ''}`}>
                    <Mail size={16} className={styles.inputIcon}/>
                    <input
                        id='email'
                        type='email'
                        className={styles.input}
                        placeholder='nombre@empresa.com'
                        autoComplete='username'
                        value={email}
                        onChange={event => setEmail(event.target.value)}
                        aria-invalid={Boolean(fieldErrors.email)}
                        required
                    />
                </div>
                {fieldErrors.email && <span className={styles.fieldError}>Ingresa un correo válido.</span>}
            </div>

            <div className={styles.field}>
                <label htmlFor='password' className={styles.label}>Contraseña</label>
                <div className={`${styles.inputWrapper} ${fieldErrors.password ? styles.invalid : ''}`}>
                    <Lock size={16} className={styles.inputIcon}/>
                    <input
                        id='password'
                        type={showPassword ? 'text' : 'password'}
                        className={styles.input}
                        placeholder='Ingresa tu contraseña'
                        autoComplete='current-password'
                        value={password}
                        onChange={event => setPassword(event.target.value)}
                        aria-invalid={Boolean(fieldErrors.password)}
                        required
                    />
                    <button
                        type='button'
                        className={styles.toggle}
                        onClick={() => setShowPassword(previous => !previous)}
                        aria-label={showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'}
                    >
                        {showPassword ? <EyeOff size={16}/> : <Eye size={16}/>}
                    </button>
                </div>
                {fieldErrors.password && <span className={styles.fieldError}>La contraseña es obligatoria.</span>}
            </div>

            <button type='submit' className={styles.submit} disabled={isSubmitting}>
                {isSubmitting
                    ? <><Loader2 size={16} className={styles.spinner}/> Verificando credenciales…</>
                    : <><LogIn size={16}/> Ingresar</>}
            </button>
        </form>
    );
}

export default LoginForm;
