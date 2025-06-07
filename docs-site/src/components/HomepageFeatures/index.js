import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

const FeatureList = [
  {
    title: 'Multi-Package Manager Support',
    emoji: '📦',
    description: (
      <>
        Automatically detects and works with npm, yarn, pnpm, or bun. 
        No configuration needed - Brummer adapts to your project.
      </>
    ),
  },
  {
    title: 'Intelligent Monitoring',
    emoji: '🔍',
    description: (
      <>
        Smart error detection, build event recognition, and test result parsing. 
        Brummer highlights what matters and filters out the noise.
      </>
    ),
  },
  {
    title: 'MCP Integration',
    emoji: '🔌',
    description: (
      <>
        Connect your favorite development tools like VSCode, Claude Code, and Cursor. 
        Access logs, execute commands, and monitor processes from anywhere.
      </>
    ),
  },
  {
    title: 'Real-time Process Management',
    emoji: '🚀',
    description: (
      <>
        Start, stop, and restart processes with visual status indicators. 
        Monitor multiple services simultaneously with ease.
      </>
    ),
  },
  {
    title: 'Browser Extension (Alpha)',
    emoji: '🌐',
    description: (
      <>
        Enhanced debugging with browser DevTools integration. 
        Capture console logs, network requests, and errors in one place.
      </>
    ),
  },
  {
    title: 'Developer Friendly',
    emoji: '💻',
    description: (
      <>
        Intuitive keyboard navigation, helpful shortcuts, and a clean TUI. 
        Designed by developers, for developers.
      </>
    ),
  },
];

function Feature({emoji, title, description}) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <div className={styles.featureEmoji}>{emoji}</div>
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}