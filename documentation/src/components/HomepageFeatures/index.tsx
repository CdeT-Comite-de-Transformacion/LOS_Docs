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
    title: 'Development Rules',
    to: '/docs/category/development-rules',
    description: (
      <>
        Naming conventions, project language, code documentation (JSDoc,
        GoDoc, Javadoc) and the Git-flow workflow with its branch and Pull
        Request rules.
      </>
    ),
  },
  {
    title: 'DDD Starter Kit',
    to: '/docs/category/standardization',
    description: (
      <>
        Architecture map and <code>modkit</code>, the Go project and DDD module
        generator based on templates, with binaries for Windows, Linux and
        macOS.
      </>
    ),
  },
  {
    title: 'News',
    to: '/news',
    description: (
      <>
        Team and project updates: welcomes, onboarding and important
        announcements.
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
            View section
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
          <Heading as="h2">What is LOS?</Heading>
          <p>
            <strong>LOS (Lean Origination System)</strong> is the origination
            system of the Transformation Committee. It manages the origination
            process end to end through backend services (BFF), process flows
            (BPM), document processing with OCR, centralized authentication and
            event messaging, deployed on AWS.
          </p>
          <p>
            This documentation gathers the rules, architecture and tools the
            team uses to build it.
          </p>
        </div>
        <Heading as="h2" className={styles.sectionsTitle}>
          Documentation overview
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
