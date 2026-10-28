import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type SectionItem = {
  title: string;
  to: string;
  description: ReactNode;
};

const SectionList: SectionItem[] = [
  {
    title: 'Reglas de Desarrollo',
    to: '/docs/category/reglas-de-desarrollo',
    description: (
      <>
        Convenciones de nomenclatura, idioma del proyecto, documentación de
        código (JSDoc, GoDoc, Javadoc) y el flujo de trabajo Git-flow con sus
        reglas de ramas y Pull Requests.
      </>
    ),
  },
  {
    title: 'DDD Starter Kit',
    to: '/docs/category/ddd-starter-kit',
    description: (
      <>
        Mapa de arquitectura y <code>modkit</code>, el generador de proyectos Go
        y módulos DDD a partir de plantillas, con binarios para Windows, Linux
        y macOS.
      </>
    ),
  },
  {
    title: 'News',
    to: '/news',
    description: (
      <>
        Novedades del equipo y del proyecto: bienvenidas, onboarding y avisos
        importantes.
      </>
    ),
  },
];

function Section({title, to, description}: SectionItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className={clsx('card', styles.sectionCard)}>
        <div className="card__header">
          <Heading as="h3">{title}</Heading>
        </div>
        <div className="card__body">
          <p>{description}</p>
        </div>
        <div className="card__footer">
          <Link className="button button--primary button--block" to={to}>
            Ver sección
          </Link>
        </div>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className={styles.about}>
          <Heading as="h2">¿Qué es LOS?</Heading>
          <p>
            <strong>LOS (Lean Origination System)</strong> es el sistema de
            originación del Comité de Transformación. Gestiona el proceso de
            originación de punta a punta mediante servicios backend (BFF),
            flujos de proceso (BPM), procesamiento de documentos con OCR,
            autenticación centralizada y mensajería de eventos, desplegado sobre
            AWS.
          </p>
          <p>
            Esta documentación reúne las reglas, la arquitectura y las
            herramientas que el equipo utiliza para construirlo.
          </p>
        </div>
        <Heading as="h2" className={styles.sectionsTitle}>
          Resumen de la documentación
        </Heading>
        <div className="row">
          {SectionList.map((props, idx) => (
            <Section key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
