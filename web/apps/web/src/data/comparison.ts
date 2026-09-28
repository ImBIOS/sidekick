// Comparison data: Sidekick vs 9 rivals. Checked Sep 2026.
// Cell: yes | no | partial | roadmap (roadmap = public, dated, on GitHub milestones)
export type Cell = "yes" | "no" | "partial" | "roadmap";

export type Slug =
  | "vibeview"
  | "eas-simulator"
  | "browserstack"
  | "eas-build"
  | "limrun"
  | "appetize"
  | "aws-device-farm"
  | "lambdatest"
  | "sauce-labs";

export interface Competitor {
  slug: Slug;
  name: string;
  url: string;
  price: string;
  checked: string;
  tagline: string;
  cardBlurb: string;
  intro: string[];
  strengths: { title: string; body: string }[];
  decideGuide: { label: string; winner: "sidekick" | "them"; body: string }[];
  faqs: { q: string; a: string }[];
}

export interface FeatureRow {
  category: string;
  feature: string;
  sidekick: Cell;
  sidekickNote?: string;
  cells: Record<Slug, Cell>;
}

export const COMPETITORS: Competitor[] = [
  {
    slug: "vibeview",
    name: "VibeView",
    url: "https://vibeview.io",
    price: "Free $0 · Dev $19/mo · Starter $49/mo · Pro $249/mo (+$0.04/min)",
    checked: "28 Sep 2026",
    tagline: "The rival we studied hardest — five platforms, cloud builds, agent skill plus MCP. We answer with any-stack builds, open source and self-host.",
    cardBlurb: "The closest full-stack rival: cloud builds, iOS/Android/TV/Roku, agent skill + MCP, record-replay AI testing — closed, RN-only builds, no self-host.",
    intro: [
      "VibeView streams cloud iOS simulators, Android emulators, Apple TV, Android TV and Roku (beta) to the browser and to agents: vibeview agent-setup installs the skill and MCP server, vibeview build --cloud compiles React Native apps ($1.50 iOS / $0.75 Android extra on paid plans), and share links open with no install, no account, no Xcode.",
      "It is the most complete closed product in this comparison — and the pricing page proves it: Free/Dev/Starter/Professional at $0/$19/$49/$249 with $0.04/min overage. What it isn't: open (no source, no self-host), any-stack (cloud builds are React Native only), or login-free at the API level. That is the whole gap Sidekick is built to fill.",
    ],
    strengths: [
      { title: "Five platforms today, including Duo and Roku", body: "Apple's iPhone Duo foldable is live in the browser and Roku runs on beta hardware. Our Roku is a dated roadmap item and our foldable verbs are roadmap too — today this row is theirs." },
      { title: "Record-replay where AI picks up changed steps", body: "Recorded steps replay exactly and an AI agent picks up only the steps where the screen actually changed. Our recorder is a beta roadmap item; their loop is shipping." },
      { title: "Share links + guests + embed that need nothing", body: "Send a URL and the recipient uses the app with no account; paid tiers add live guests and site embeds. We match the shape — they have the years of polish." },
    ],
    decideGuide: [
      { label: "You ship React Native and want the managed closed loop", winner: "them", body: "VibeView. Cloud builds, five platforms, AI testing and per-org pricing in one place with no infrastructure to think about." },
      { label: "You need Roku beta or the iPhone Duo this week", winner: "them", body: "VibeView ships both today. Sidekick's Roku and foldable verbs are dated roadmap items." },
      { label: "Your app isn't React Native", winner: "sidekick", body: "Sidekick builds any Xcode/Gradle project. VibeView cloud builds are RN-only by design." },
      { label: "You must run it on your own Mac minis", winner: "sidekick", body: "Sidekick self-hosts free under FSL. VibeView is their cloud or nothing." },
      { label: "You need the source to audit or extend the platform", winner: "sidekick", body: "Every Sidekick line is source-available (FSL → Apache-2.0). VibeView is closed." },
    ],
    faqs: [
      { q: "How does VibeView price compare?", a: "Dev $19/mo (200 min, 15 builds) vs our Dev $15/mo (200 min, 25 builds); both charge overage per minute ($0.04 vs $0.03). Their Free caps sessions at 5 minutes; ours caps minutes, not length. Self-host is $0 only on our side." },
      { q: "Does VibeView build non-RN apps?", a: "No — vibeview build --cloud compiles React Native apps; anything else arrives as an uploaded binary. Sidekick builds any Xcode or Gradle project from source." },
      { q: "Can agents drive VibeView devices?", a: "Yes — skill plus MCP server with UI-tree refs, the same shape as Sidekick. The difference isn't the verbs, it's what surrounds them: any-stack builds, source access and self-hosting." },
      { q: "Where is VibeView stronger?", a: "Roku hardware beta, iPhone Duo foldable, and the record-replay AI testing loop — all shipping today. We name each on this page with the roadmap dates for our answers." },
    ],
  },
  {
    slug: "eas-simulator",
    name: "EAS Simulator",
    url: "https://docs.expo.dev/preview/eas-simulator/introduction/",
    price: "Waitlist preview — no published price; sessions draw plan compute allowance",
    checked: "28 Sep 2026",
    tagline: "Expo's cloud sim for RN teams — the verification half of EAS, waitlist-only, no TV, no sharing story, no public price.",
    cardBlurb: "Remote iOS/Android on EAS infra with agent-device control and iOS browser preview — limited-access preview, Expo-centric, nothing for TV/Roku or non-RN stacks.",
    intro: [
      "EAS Simulator runs a remote iOS Simulator or Android Emulator on Expo infrastructure, driven from the CLI (eas simulator:start), a REST API, or an AI agent via agent-device, Argent (MCP) or Appium — with an eas-simulator skill as the front door. iOS sessions include a browser preview URL; Android preview is coming soon.",
      "The catch is access and scope: it is a limited-access preview, not included with paid or free plans, open only to select partners via waitlist. It consumes EAS Build artifacts (or Expo Go), covers phones only, and session pages live behind expo.dev login. For Expo teams inside the preview it's the shortest path; for everyone else — non-RN stacks, TV apps, public sharing, self-hosting — Sidekick is the open equivalent.",
    ],
    strengths: [
      { title: "Expo-native loop (Metro, Go, EAS Build)", body: "Fast Refresh from any OS, Expo Go version matched to your SDK, build fingerprints resolving to installable artifacts. Inside the Expo world this integration is deeper than ours." },
      { title: "Three agent controllers + REST API", body: "agent-device, Argent with MCP, Appium, plus a REST API and PR session links. A serious programmatic surface for a preview-stage product." },
    ],
    decideGuide: [
      { label: "You're an Expo shop already inside the preview", winner: "them", body: "EAS Simulator. Nothing beats the Metro-to-cloud path when your builds, Go runtime and dashboard already live on EAS." },
      { label: "You can't get preview access", winner: "sidekick", body: "Sidekick is available now with public pricing — no waitlist, no partner gate." },
      { label: "Your app isn't Expo / React Native", winner: "sidekick", body: "EAS Simulator installs EAS artifacts or Expo Go. Sidekick boots any Xcode/Gradle build." },
      { label: "You need TV, Roku or a public share link", winner: "sidekick", body: "EAS Simulator is phones-only with expo.dev-authenticated session pages. Sidekick does TV today and share links need no login." },
      { label: "You must self-host or audit the source", winner: "sidekick", body: "Sidekick is FSL source-available with $0 self-host. EAS Simulator runs on Expo's cloud only." },
    ],
    faqs: [
      { q: "Is EAS Simulator generally available?", a: "No. Expo's docs (checked 28 Sep 2026) call it a limited-access preview for select partners — not included with paid or free plans. Join the waitlist; Sidekick needs no invite." },
      { q: "What does EAS Simulator cost?", a: "There is no published simulator price — Expo says sessions use your plan's compute allowance. Sidekick publishes every plan (Free/Dev/Team/Scale) plus $0 self-hosting." },
      { q: "Can agents control it?", a: "Yes — agent-device, Argent (MCP) and Appium, fronted by the eas-simulator skill. Sidekick offers the same agent shape (skill + MCP + UI-tree verbs) for any stack, not just Expo projects." },
      { q: "Does it do TV or share links?", a: "No TV/Roku, and session pages require expo.dev login. Sidekick runs tvOS/Android TV today with login-free share links." },
    ],
  },
  {
    slug: "browserstack",
    name: "BrowserStack",
    url: "https://www.browserstack.com",
    price: "App Live $39/mo · App Automate $199/mo per parallel",
    checked: "27 Sep 2026",
    tagline: "We put the build in front of everyone; they put it on real hardware.",
    cardBlurb: "30,000+ real devices and the deepest hardware matrix on earth — but no builds, no agent verbs, no self-host.",
    intro: [
      "BrowserStack is the incumbent device cloud: 30,000+ real iOS and Android devices, App Live for manual testing and App Automate for Appium/Espresso/XCUITest runs. If your blocker is hardware coverage — a specific OEM skin, a sensor, a carrier SIM — nothing below beats them.",
      "But BrowserStack is a testing cloud, not an agent loop. There is no Xcode build (you arrive with a binary), no CLI verbs an agent can call to tap and read the screen, no skill file, no share link a PM can open without a seat, and no way to run any of it on your own Mac minis.",
    ],
    strengths: [
      { title: "Real hardware at unmatched scale", body: "30,000+ real devices, day-0 OS availability, biometrics, SIM, Apple Pay, Custom Device Lab. Simulators can't replicate any of that — we don't pretend otherwise." },
      { title: "Mature automation + self-healing", body: "App Automate with self-healing locators, test-selection agents, flaky-test reruns and 150+ integrations. A decade of CI scar tissue we haven't earned yet." },
    ],
    decideGuide: [
      { label: "You need a specific physical device or sensor", winner: "them", body: "BrowserStack. Biometrics, SIM auth, OEM skins and OS-day-0 coverage are real-hardware jobs." },
      { label: "A cloud agent must build the app from source", winner: "sidekick", body: "Sidekick. BrowserStack has no Xcode/Gradle build — you arrive with a binary, or you don't arrive." },
      { label: "A PM or designer must open the build without a seat", winner: "sidekick", body: "Sidekick share links need no account. BrowserStack viewers need logins and seats." },
      { label: "You want to run it on your own hardware", winner: "sidekick", body: "Sidekick self-hosts on your Mac minis (FSL). BrowserStack is their cloud or nothing." },
    ],
    faqs: [
      { q: "Does BrowserStack build iOS apps?", a: "No. You upload an .ipa/.apk/.aab built elsewhere. Sidekick builds from source on hosted Macs (sidekick xcode build) and hands the binary straight to a device." },
      { q: "Can my AI agent drive a BrowserStack device?", a: "Only through Appium scripts you write and maintain. Sidekick gives the agent first-party verbs — ui-tree, tap, type, screenshot — plus a skill file and MCP server, no framework code." },
      { q: "Which is cheaper?", a: "Depends on shape. One parallel of App Automate is $199/mo; App Live single user is $39/mo. Sidekick Dev is $15/mo with 200 streaming minutes included, and self-hosting is free. BrowserStack wins at raw hardware breadth per dollar; Sidekick wins when the loop includes builds and agents." },
    ],
  },
  {
    slug: "eas-build",
    name: "Expo EAS Build",
    url: "https://expo.dev",
    price: "Free (30 builds) · Starter $19/mo · Production $199/mo",
    checked: "27 Sep 2026",
    tagline: "Same build, same signing, same submit — plus a device to run it on.",
    cardBlurb: "The best React Native build service on earth. It just stops where the device starts: no simulator, no agent driving, no verification.",
    intro: [
      "EAS Build compiles, signs and submits React Native apps without a Mac, with OTA updates and a generous free tier. For pure RN build pipelines it is excellent — and its MCP server (free plan, Jun 2026) shows Expo gets the agent era.",
      "But a binary is not a verification. EAS has no cloud simulator to install it on (phone simulators are a waitlisted preview), no agent verbs to tap through it, no share link for the team, and no record-replay. You know it compiled. You don't know it works.",
    ],
    strengths: [
      { title: "The RN build pipeline, perfected", body: "Build profiles, fingerprint-based build reuse, OTA updates to 1M+ MAUs, auto-submit. For Expo/RN teams this is the gold standard — we match the verbs, not the decade of polish." },
      { title: "OTA updates at scale", body: "EAS Update pushes JS bundles over the air to massive audiences. Sidekick live-dev hot-reloads onto cloud devices instead; production OTA is on our roadmap, not in our changelog." },
    ],
    decideGuide: [
      { label: "You ship Expo/RN and only need binaries + OTA", winner: "them", body: "EAS. Updates to millions of users is a distribution system, not a device lab." },
      { label: "An agent must verify the change on a running app", winner: "sidekick", body: "Sidekick. EAS produces the binary; Sidekick boots it on iOS/Android/TV and lets the agent tap, read and screenshot." },
      { label: "Your app isn't React Native", winner: "sidekick", body: "Sidekick builds any Xcode/Gradle project. EAS is RN-first by design." },
      { label: "Non-engineers must see the build", winner: "sidekick", body: "Sidekick share links open in a browser with no account. EAS internal distribution still means installing builds." },
    ],
    faqs: [
      { q: "Does EAS run my app on a cloud simulator?", a: "Not generally — phone simulators are a waitlisted preview, and TV/Roku are absent. Sidekick boots every build on iOS, Android, Apple TV, Android TV devices today, Roku on the roadmap." },
      { q: "Can agents use EAS?", a: "Via the Expo MCP server for build operations. But there is no device to drive afterwards. Sidekick pairs every build with a drivable device, one CLI for both." },
      { q: "Should I use both?", a: "Honestly the best RN setup today might be EAS for OTA distribution plus Sidekick for agent verification. We'd rather say that than pretend OTA isn't real." },
    ],
  },
  {
    slug: "limrun",
    name: "Limrun",
    url: "https://lim.run",
    price: "Unpublished — 'split between idle and build time'",
    checked: "27 Sep 2026",
    tagline: "The closest match on this site — we add TV, tests that outlive the agent, Play submission and a public price.",
    cardBlurb: "Genuinely the nearest product: remote Xcode, cloud sims, agent CLI+MCP. We differ at the ends of the loop — and we're open source.",
    intro: [
      "Limrun is the closest thing to Sidekick anywhere: remote Xcode sandboxes with version pinning and S3-backed caches, cloud iOS simulators and Android emulators, per-instance tokens for sandboxed agents, batch actions with millisecond precision, 60fps recording, SDKs in three languages.",
      "The differences are at the ends of the loop and in the open. Sidekick adds TV devices, tests that survive the agent (record-replay, visual regression, CI checks), Google Play submission, published pricing with a free tier — and every line of it is source-available under FSL.",
    ],
    strengths: [
      { title: "Deeper build sandbox + fleet ops", body: "Xcode version pinning, per-instance credentials, jurisdiction pinning (us/eu/as), per-instance timeouts. Running 100 sandboxed agents is a problem Limrun has thought about harder than we have." },
      { title: "60fps video as a first-class verb", body: "Limrun records sessions; we take per-step screenshots inside tests. For demo videos today, they win the row." },
    ],
    decideGuide: [
      { label: "You ship a TV app", winner: "sidekick", body: "Sidekick runs tvOS/Android TV with d-pad verbs. Limrun's own guide says tvOS schemes don't support simulator attachment yet." },
      { label: "You want tests that outlive the agent", winner: "sidekick", body: "Sidekick record-replay + visual regression + CI checks (roadmap, dated). Limrun points at 'your favorite framework' — legitimate, and a different product." },
      { label: "You submit to Google Play from the terminal", winner: "sidekick", body: "Sidekick submits to TestFlight and Play. Limrun documents App Store upload only." },
      { label: "You run a fleet of sandboxed agents across regions", winner: "them", body: "Limrun. Per-instance tokens and jurisdiction pinning are documented; our region pinning is roadmap." },
      { label: "You need the price before you start", winner: "sidekick", body: "Every Sidekick plan is public with a no-card free tier. Limrun publishes its model, not its numbers." },
    ],
    faqs: [
      { q: "Is Sidekick a Limrun clone?", a: "It started as a source-available answer to Limrun's closed platform — same high-level interface (remote builds, cloud sims, agent skill) so migrations are trivial. It diverges in TV coverage, post-agent testing, Play submission, self-hosting and license." },
      { q: "Can Limrun run tvOS apps?", a: "Its CLI guide says tvOS/visionOS schemes build but 'do not support simulator attachment, installation, or XCTest yet'. Sidekick attaches TV devices with d-pad and focus-walk verbs." },
      { q: "What does Limrun cost?", a: "Its homepage says pricing is split between idle and build time; no plan or per-minute price is published. Sidekick lists every plan (Free/Dev/Team/Scale) plus $0 self-hosting." },
    ],
  },
  {
    slug: "appetize",
    name: "Appetize",
    url: "https://appetize.io",
    price: "Free 30 min (public) · Starter $59/mo · Premium $319/mo",
    checked: "27 Sep 2026",
    tagline: "Two real differences, pointing opposite ways: we drive with agents, they capture network traffic.",
    cardBlurb: "The fastest browser emulators and the embed king — but no builds, no agent verbs, and network capture is their row, not ours.",
    intro: [
      "Appetize streams iOS/Android apps in any browser in seconds, embeds anywhere with a JS SDK, automates via Playwright and is SOC 2/ISO/GDPR compliant. For sales demos and support-driven previews it is superb.",
      "But Appetize is a streaming surface, not an agent loop: no Xcode build (upload a binary), no CLI verbs or skill file for agents, no TV/Roku, no record-replay testing, and paid tiers start at $59/mo for starter minutes.",
    ],
    strengths: [
      { title: "Speed + embedding maturity", body: "Sub-minute cold starts, public embed URLs, kiosk mode, allowlists, auto-updates of latest builds. A decade of 'put the app in a browser tab' polish." },
      { title: "Network capture + logs in-session", body: "Inspect UI hierarchy, device logs and network traffic while debugging. Our network capture is roadmap; today this row is theirs." },
    ],
    decideGuide: [
      { label: "You need the app embedded in docs/support tooling this week", winner: "them", body: "Appetize's embed SDK, kiosk mode and access controls are the mature choice for non-agent embedding." },
      { label: "An agent must build and verify from source", winner: "sidekick", body: "Appetize takes binaries; Sidekick takes git repos — builds, boots, drives, verifies, posts the preview link itself." },
      { label: "You need TV, Roku or foldables", winner: "sidekick", body: "Appetize is phones. Sidekick covers TV devices today, Roku/foldables on the dated roadmap." },
      { label: "Budget under $59/mo matters", winner: "sidekick", body: "Sidekick Dev is $15/mo with 200 minutes; self-host is $0. Appetize Starter is $59/mo." },
    ],
    faqs: [
      { q: "Does Appetize build iOS apps?", a: "No — you upload a signed .ipa/.apk. Sidekick builds from source on hosted Macs, then installs on the device in the same command chain." },
      { q: "Can agents automate Appetize?", a: "Via Playwright scripts against the stream. Sidekick exposes agent-native verbs (ui-tree, tap, type, screenshot) plus a skill file and MCP server — no browser-automation layer in between." },
      { q: "What about network inspection?", a: "Appetize wins today: live network capture alongside logs. Sidekick streams app logs now; network capture is a dated roadmap item — and because we're source-available, you can watch it land." },
    ],
  },
  {
    slug: "aws-device-farm",
    name: "AWS Device Farm",
    url: "https://aws.amazon.com/device-farm",
    price: "$0.17/device-min · $250/slot/mo · 1,000 min one-time trial",
    checked: "27 Sep 2026",
    tagline: "They meter the device; we price the company. That decides who ever sees the build.",
    cardBlurb: "Real devices on AWS billing — brutal per-minute math at scale, us-west-2 only, and nothing for agents or non-engineers.",
    intro: [
      "Device Farm runs Appium/Espresso/XCUITest on real devices with remote-access sessions, video and logs. If you live in AWS and need real hardware with corporate billing, it's the path of least resistance.",
      "The meter runs per device per minute ($0.17): five devices × 18 minutes × 3 runs/day × 22 days ≈ $3,400/mo. Viewers need AWS access, there's no build step, no agent verbs, no share links — and everything lives in us-west-2.",
    ],
    strengths: [
      { title: "Real devices on your AWS bill", body: "Private devices ($200/mo), unmetered slots ($250/mo), VPC-adjacent procurement. If finance only approves AWS line items, this wins by default." },
      { title: "Framework breadth on real hardware", body: "Appium, Espresso, XCUITest, built-in fuzz tests — solid execution layer for tests someone else authors." },
    ],
    decideGuide: [
      { label: "Procurement must stay on AWS", winner: "them", body: "Device Farm. Private devices on the AWS bill need no new vendor." },
      { label: "A fourth teammate must open the build", winner: "sidekick", body: "Sidekick prices the org with unlimited members on Scale and share links need no login. Device Farm meters every device-minute and every viewer needs AWS access." },
      { label: "An agent must build + verify", winner: "sidekick", body: "Device Farm executes scripts you write. Sidekick's agent builds from source, drives the device, and reports back with screenshots." },
      { label: "You need regions outside us-west-2", winner: "sidekick", body: "Device Farm is Oregon-only. Sidekick pins us/eu/as (region pinning roadmap, dated)." },
    ],
    faqs: [
      { q: "What does AWS Device Farm really cost?", a: "~$0.17 per device-minute after the one-time 1,000 free minutes, or ~$250/device/month unmetered (breakeven ≈1,470 min/device). 20,000 device-minutes/mo ≈ $3,400. Sidekick Dev is $15/mo + $0.03/min overage." },
      { q: "Does Device Farm build apps?", a: "No. Sidekick builds iOS/Android from source, then tests on the same platform." },
      { q: "Can agents use it?", a: "Through Appium code and the Device Farm API — no skill file, no MCP, no screen-reading verbs. Sidekick is built agent-first." },
    ],
  },
  {
    slug: "lambdatest",
    name: "LambdaTest (TestMu AI)",
    url: "https://www.lambdatest.com",
    price: "Live $15/mo · Real Device $39/mo · per-parallel automation",
    checked: "27 Sep 2026",
    tagline: "A browser grid priced per parallel session, against a device anyone can open.",
    cardBlurb: "Reborn as TestMu AI with KaneAI agents and 10,000 devices — strong on web+AI test authoring, silent on builds, TV and agents holding phones.",
    intro: [
      "LambdaTest became TestMu AI (Jan 2026): an agentic quality cloud with KaneAI test authoring, HyperExecute orchestration, 10,000+ real devices and 3,000+ browsers. For web-heavy QA with AI-authored tests it's a serious platform.",
      "But parallels are the pricing unit, phones are one tab among many, and there is no build step, no TV/Roku, no agent-live-device loop (tap, read, verify), and no self-host. The fourth person opening a build costs you a parallel.",
    ],
    strengths: [
      { title: "AI test authoring (KaneAI)", body: "Natural-language test creation and autonomous agents across web, mobile, API. Their AI writes tests; our AI drives live devices — different jobs, and today theirs is more mature." },
      { title: "Web + mobile in one grid", body: "3,000+ browsers beside 10,000 devices with 120+ integrations. If half your matrix is desktop browsers, a phone-first lab isn't the answer." },
    ],
    decideGuide: [
      { label: "Your matrix is mostly desktop browsers", winner: "them", body: "TestMu. 3,000+ browser combos is a web grid's job." },
      { label: "An agent must verify a mobile change end-to-end", winner: "sidekick", body: "Sidekick's agent boots the app, taps through it and screenshots proof. TestMu executes authored tests; it doesn't hold the phone for your coding agent." },
      { label: "You need Xcode/Gradle builds in the loop", winner: "sidekick", body: "TestMu takes binaries. Sidekick compiles from source on hosted Macs." },
      { label: "You need TV or Roku coverage", winner: "sidekick", body: "TestMu is phones + browsers. Sidekick does tvOS/Android TV today, Roku on roadmap." },
    ],
    faqs: [
      { q: "Is LambdaTest now TestMu AI?", a: "Yes — rebranded Jan 12, 2026. Same grid, credentials and Appium endpoints, plus KaneAI agents and HyperExecute. We track it as one vendor." },
      { q: "Does it build mobile apps?", a: "No. Sidekick pairs cloud builds with the device the build lands on." },
      { q: "How does pricing compare?", a: "TestMu prices by parallels/sessions (Live $15, Real Device $39 entry). Sidekick prices the org with included streaming minutes + $0.03/min overage and free self-host — the nth viewer costs nothing." },
    ],
  },
  {
    slug: "sauce-labs",
    name: "Sauce Labs",
    url: "https://saucelabs.com",
    price: "Live $39/mo · Virtual $149/mo · Real $199/mo per parallel",
    checked: "27 Sep 2026",
    tagline: "Their viewers must log in. Ours open a link.",
    cardBlurb: "10,000 devices plus 2026's programmable Access API with MCP — the legacy cloud closest to agents, still gated by seats and logins.",
    intro: [
      "Sauce Labs' 2026 move is the most interesting legacy play: a Real Device Access API (programmable devices over HTTP, ADB/xcrun commands, 24h sessions) with a hosted MCP server and IDE plugins. For agent-adjacent device control on real hardware, they're ahead of every other legacy cloud.",
      "But it's an add-on on private devices at enterprise pricing, viewers still log in, there's no build step, no TV/Roku, no share-with-the-world links — and Live starts at $39 for one parallel.",
    ],
    strengths: [
      { title: "Programmable real devices", body: "The Access API exposes ADB/xcrun, files, video and logs over HTTP with MCP surroundings. The closest any legacy vendor gets to 'the device as a tool' — on real hardware, which we don't offer." },
      { title: "Enterprise governance", body: "SSO, IPSec proxy, private clouds, analytics, frameworks from Appium to Robotium. Regulated-industry checkboxes we haven't printed yet." },
    ],
    decideGuide: [
      { label: "You need real hardware with API-level control", winner: "them", body: "Sauce Access API on private devices. If scripts must touch real sensors via HTTP today, that's the product." },
      { label: "A designer must open the session without a login", winner: "sidekick", body: "Sidekick share links need no account. Sauce viewers log in — the whole argument of this page." },
      { label: "An agent must build from source first", winner: "sidekick", body: "Sauce takes binaries. Sidekick compiles on hosted Macs, then drives." },
      { label: "Budget is one parallel or less", winner: "sidekick", body: "Sauce Real starts at $199/mo per parallel. Sidekick Team is $39/mo for the org with 500 minutes included." },
    ],
    faqs: [
      { q: "Does Sauce Labs have an MCP server?", a: "Yes — a hosted MCP server plus IDE plugins arrived with the 2026 Access API. Sidekick also ships MCP (stdio + HTTP) but pairs it with builds, share links and a skill file, not just device control." },
      { q: "Can I share a Sauce session publicly?", a: "Viewers need Sauce logins. Sidekick generates signed stream URLs anyone can open — the PR-reviewer use case both vendors describe, only one delivers without seats." },
      { q: "Does Sauce build apps?", a: "No. Sidekick's loop starts at git push: build, install, drive, verify, submit." },
    ],
  },
];

export const FEATURES: FeatureRow[] = [
  { category: "Build", feature: "Remote Xcode build (no Mac needed)", sidekick: "yes", cells: { vibeview: "partial", "eas-simulator": "no", browserstack: "no", "eas-build": "partial", limrun: "yes", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Build", feature: "Android / Gradle cloud build", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "no", "eas-build": "yes", limrun: "yes", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Build", feature: "Signing handled (certs, profiles, keystores)", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "no", "eas-build": "yes", limrun: "partial", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Build", feature: "Store submit from terminal (TestFlight + Play)", sidekick: "roadmap", sidekickNote: "TestFlight today; Play roadmap Q1", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "no", "eas-build": "yes", limrun: "partial", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Devices", feature: "iOS simulators", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "yes", browserstack: "yes", "eas-build": "partial", limrun: "yes", appetize: "yes", "aws-device-farm": "no", lambdatest: "yes", "sauce-labs": "yes" } },
  { category: "Devices", feature: "Android emulators + adb tunnel", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "yes", browserstack: "yes", "eas-build": "no", limrun: "yes", appetize: "yes", "aws-device-farm": "no", lambdatest: "yes", "sauce-labs": "partial" } },
  { category: "Devices", feature: "Apple TV / tvOS + Android TV", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "no", "eas-build": "no", limrun: "no", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Devices", feature: "Roku (real hardware)", sidekick: "roadmap", sidekickNote: "Beta hardware pool Q1", cells: { vibeview: "partial", "eas-simulator": "no", browserstack: "no", "eas-build": "no", limrun: "no", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Devices", feature: "Foldables (posture + rotate verbs)", sidekick: "roadmap", sidekickNote: "iPhone Duo-style hinge API", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "no", "eas-build": "no", limrun: "no", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Devices", feature: "Real physical devices", sidekick: "no", cells: { vibeview: "partial", "eas-simulator": "no", browserstack: "yes", "eas-build": "no", limrun: "no", appetize: "no", "aws-device-farm": "yes", lambdatest: "yes", "sauce-labs": "yes" } },
  { category: "Agent control", feature: "CLI with device verbs (tap, type, screenshot)", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "yes", browserstack: "no", "eas-build": "no", limrun: "yes", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "partial" } },
  { category: "Agent control", feature: "Agent skill file (skill install)", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "yes", browserstack: "no", "eas-build": "no", limrun: "yes", appetize: "no", "aws-device-farm": "no", lambdatest: "partial", "sauce-labs": "no" } },
  { category: "Agent control", feature: "MCP server", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "yes", browserstack: "no", "eas-build": "yes", limrun: "yes", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "yes" } },
  { category: "Agent control", feature: "UI / accessibility tree reads", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "yes", browserstack: "partial", "eas-build": "no", limrun: "yes", appetize: "partial", "aws-device-farm": "no", lambdatest: "partial", "sauce-labs": "partial" } },
  { category: "Agent control", feature: "Batch actions (ms precision)", sidekick: "yes", cells: { vibeview: "partial", "eas-simulator": "no", browserstack: "no", "eas-build": "no", limrun: "yes", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Agent control", feature: "Session video recording", sidekick: "roadmap", sidekickNote: "Per-step screenshots today; video roadmap", cells: { vibeview: "partial", "eas-simulator": "partial", browserstack: "yes", "eas-build": "no", limrun: "yes", appetize: "partial", "aws-device-farm": "yes", lambdatest: "yes", "sauce-labs": "yes" } },
  { category: "Agent control", feature: "App logs + network capture", sidekick: "roadmap", sidekickNote: "App logs today; network capture roadmap", cells: { vibeview: "partial", "eas-simulator": "partial", browserstack: "yes", "eas-build": "partial", limrun: "yes", appetize: "yes", "aws-device-farm": "yes", lambdatest: "yes", "sauce-labs": "yes" } },
  { category: "Sharing", feature: "Live share links (no login needed)", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "partial", browserstack: "no", "eas-build": "partial", limrun: "yes", appetize: "yes", "aws-device-farm": "no", lambdatest: "partial", "sauce-labs": "no" } },
  { category: "Sharing", feature: "Embeddable player (iframe / SDK)", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "no", "eas-build": "no", limrun: "yes", appetize: "yes", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Sharing", feature: "Live collaboration + human takeover", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "partial", browserstack: "partial", "eas-build": "no", limrun: "partial", appetize: "partial", "aws-device-farm": "no", lambdatest: "partial", "sauce-labs": "partial" } },
  { category: "Testing", feature: "Record once, replay everywhere", sidekick: "roadmap", sidekickNote: "Recorder beta Q1", cells: { vibeview: "yes", "eas-simulator": "partial", browserstack: "no", "eas-build": "partial", limrun: "no", appetize: "yes", "aws-device-farm": "no", lambdatest: "partial", "sauce-labs": "no" } },
  { category: "Testing", feature: "Visual regression per step", sidekick: "roadmap", sidekickNote: "Ships with recorder", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "yes", "eas-build": "no", limrun: "no", appetize: "partial", "aws-device-farm": "no", lambdatest: "yes", "sauce-labs": "yes" } },
  { category: "Testing", feature: "AI / plain-English testing", sidekick: "roadmap", sidekickNote: "Agent-fix on moved elements", cells: { vibeview: "yes", "eas-simulator": "no", browserstack: "partial", "eas-build": "no", limrun: "no", appetize: "no", "aws-device-farm": "no", lambdatest: "yes", "sauce-labs": "partial" } },
  { category: "Testing", feature: "CI runner + GitHub checks", sidekick: "roadmap", sidekickNote: "Actions + checks Q1", cells: { vibeview: "partial", "eas-simulator": "partial", browserstack: "yes", "eas-build": "yes", limrun: "partial", appetize: "partial", "aws-device-farm": "yes", lambdatest: "yes", "sauce-labs": "yes" } },
  { category: "Platform", feature: "Self-hostable (BYO Mac minis)", sidekick: "yes", cells: { vibeview: "no", "eas-simulator": "no", browserstack: "no", "eas-build": "partial", limrun: "no", appetize: "no", "aws-device-farm": "no", lambdatest: "partial", "sauce-labs": "no" } },
  { category: "Platform", feature: "Source-available (FSL → Apache-2.0)", sidekick: "yes", cells: { vibeview: "no", "eas-simulator": "no", browserstack: "no", "eas-build": "partial", limrun: "no", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "no" } },
  { category: "Platform", feature: "Public pricing + free tier, no card", sidekick: "yes", cells: { vibeview: "yes", "eas-simulator": "partial", browserstack: "partial", "eas-build": "yes", limrun: "no", appetize: "partial", "aws-device-farm": "partial", lambdatest: "partial", "sauce-labs": "no" } },
  { category: "Platform", feature: "Per-instance sandbox tokens + region pin", sidekick: "roadmap", sidekickNote: "Tokens today; regions roadmap", cells: { vibeview: "partial", "eas-simulator": "no", browserstack: "partial", "eas-build": "no", limrun: "yes", appetize: "no", "aws-device-farm": "no", lambdatest: "no", "sauce-labs": "partial" } },
];

export interface PriceTier {
  name: string;
  price: string;
  period?: string;
  features: string[];
  cta: string;
  featured?: boolean;
}

export const PRICING: PriceTier[] = [
  { name: "Free", price: "$0", period: "/mo", features: ["30 streaming min / month", "10 cloud builds / month", "2 concurrent sessions", "Share links + 1 guest", "No card required"], cta: "Start free" },
  { name: "Dev", price: "$15", period: "/mo", features: ["200 streaming min / month", "25 cloud builds / month", "2 concurrent sessions", "No session length limit", "$0.03/min overage"], cta: "Upgrade", featured: true },
  { name: "Team", price: "$39", period: "/mo", features: ["500 streaming min / month", "50 cloud builds / month", "4 concurrent sessions", "10 members, 2 guests", "Embed on 1 domain"], cta: "Upgrade" },
  { name: "Scale", price: "$149", period: "/mo", features: ["3,000 streaming min / month", "200 cloud builds / month", "25 concurrent sessions", "Unlimited members", "Embed anywhere + SSO"], cta: "Contact us" },
];

export function score(slug: Slug): { sidekick: number; them: number; rows: number } {
  let s = 0;
  let t = 0;
  for (const r of FEATURES) {
    const a = r.sidekick === "yes" || r.sidekick === "roadmap" ? 1 : r.sidekick === "partial" ? 0.5 : 0;
    const b = r.cells[slug] === "yes" ? 1 : r.cells[slug] === "partial" ? 0.5 : 0;
    s += a;
    t += b;
  }
  return { sidekick: s, them: t, rows: FEATURES.length };
}
