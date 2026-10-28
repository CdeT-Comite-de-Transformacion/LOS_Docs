import type { ReactNode } from "react";
import clsx from "clsx";
import Link from "@docusaurus/Link";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

type SectionItem = {
  title: string;
  to: string;
  description: ReactNode;
};

const SectionList: SectionItem[] = [
  {
    title: "Documentation",
    to: "/docs/category/development-rules",
    description: (
      <>
        The project standards: naming conventions, project language, code
        documentation and the Git-flow workflow, plus the architecture map and
        the <code>modkit</code> generator.
      </>
    ),
  },
  {
    title: "Projects",
    to: "/projects",
    description: (
      <>
        The projects currently in the area, starting with LOS (Lean Origination
        System), with their frontend and backend documentation.
      </>
    ),
  },
  {
    title: "News",
    to: "/news",
    description: (
      <>
        Team and project news: welcomes, onboarding and important announcements.
      </>
    ),
  },
];

function Section({ title, to, description }: SectionItem) {
  return (
    <div className={clsx("col col--4")}>
      <div className={clsx("card", styles.sectionCard)}>
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
          <Heading as="h2">Official documentation space</Heading>
          <p>
            This is the official documentation space of the Transformation
            Committee. Here you will find the standards that define how we build
            software, the projects currently under development in the area, and
            the latest news from the team and its projects.
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
