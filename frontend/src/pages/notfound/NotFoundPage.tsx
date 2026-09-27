import styles from './notfound.module.css';
import {Link} from 'react-router';
import {Compass} from 'lucide-react';

const NotFoundPage = () =>
{
    return (
        <section className={styles.container}>
            <div className={styles.icon}>
                <Compass size={32} strokeWidth={1.5}/>
            </div>
            <span className={styles.code}>404</span>
            <h1 className={styles.title}>Esta página no existe</h1>
            <p className={styles.description}>La ruta que buscas no está disponible o fue movida.</p>
            <Link to='/dashboard' className={styles.action}>Volver al dashboard</Link>
        </section>
    );
}

export default NotFoundPage;
