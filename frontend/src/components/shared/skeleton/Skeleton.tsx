import styles from './skeleton.module.css';

interface SkeletonProps {
    width?: string;
    height?: string;
    radius?: string;
}

const Skeleton = ({width = '100%', height = '16px', radius = 'var(--radius-sm)'}: SkeletonProps) =>
{
    return <span className={styles.skeleton} style={{width, height, borderRadius: radius}} aria-hidden='true'/>;
}

export default Skeleton;
