-- Seed data: demonstration skills for development
-- This runs after goose migrations on first boot

-- Create a demo namespace
INSERT INTO namespaces (id, name, type) VALUES
    ('00000000-0000-0000-0000-000000000001', 'skillhub-demo', 'personal')
ON CONFLICT (name) DO NOTHING;

-- Create a demo token for the namespace
INSERT INTO tokens (id, namespace_id, token_hash, label) VALUES
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001',
     'seed_demo_token_do_not_use_in_production', 'seed-data')
ON CONFLICT (token_hash) DO NOTHING;

-- Create demo skills
INSERT INTO skills (id, namespace_id, name, description, tags, framework, visibility, install_count, avg_rating, rating_count, outcome_success_rate, latest_version, status) VALUES
    ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001',
     'code-review', 'Reviews code for security vulnerabilities, performance issues, and best practices. Returns structured findings with severity levels P0-P3.',
     ARRAY['security', 'code-quality', 'review'], 'claude-code', 'public', 42, 8.50, 12, 0.870, '1.0.0', 'active'),
    ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001',
     'docker-deploy', 'Builds and deploys containerized applications. Handles multi-stage Dockerfile generation, compose orchestration, and registry push.',
     ARRAY['deployment', 'docker', 'devops'], 'gstack', 'public', 28, 7.80, 8, 0.920, '1.0.0', 'active'),
    ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000001',
     'git-workflow', 'Manages complex git workflows including branch strategies, merge conflict resolution, and PR descriptions.',
     ARRAY['git', 'workflow', 'automation'], 'cursor', 'public', 15, 9.10, 6, 0.950, '1.0.0', 'active'),
    ('00000000-0000-0000-0000-000000000104', '00000000-0000-0000-0000-000000000001',
     'skillhub-discovery', 'The AI SkillHub Discovery Skill. Enables agents to search, install, rate, and contribute skills from the SkillHub registry.',
     ARRAY['skillhub', 'discovery', 'search', 'meta'], 'gstack', 'public', 100, 9.50, 20, 0.980, '2.0.0', 'active')
ON CONFLICT (namespace_id, name) DO NOTHING;

-- Create demo revisions
INSERT INTO revisions (id, skill_id, version, content, change_summary, author_token_id, review_status, schema_type, triggers, estimated_tokens) VALUES
    ('00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000101',
     '1.0.0',
     E'---\nname: code-review\nversion: 1.0.0\nschema: skill-md\nframework: claude-code\ntags: [security, code-quality, review]\ndescription: "Reviews code for security vulnerabilities, performance issues, and best practices."\ntriggers: ["review code", "security audit", "code quality"]\nestimated_tokens: 1500\nrequirements:\n  tools: [read, bash]\n---\n\n# code-review\n\nYou are a code review expert. When asked to review code:\n\n1. Read the target files\n2. Analyze for: security vulnerabilities, performance issues, error handling gaps, and style violations\n3. Output findings as structured list with severity P0 (critical) through P3 (nitpick)\n4. Suggest specific fixes for each finding',
     'Initial version', '00000000-0000-0000-0000-000000000010', 'approved', 'skill-md',
     ARRAY['review code', 'security audit', 'code quality'], 1500),
    ('00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000102',
     '1.0.0',
     E'---\nname: docker-deploy\nversion: 1.0.0\nschema: skill-md\nframework: gstack\ntags: [deployment, docker, devops]\ndescription: "Builds and deploys containerized applications."\ntriggers: ["deploy", "dockerize", "containerize"]\nestimated_tokens: 1200\nrequirements:\n  tools: [bash, write]\n  software:\n    - name: docker\n      check_command: "docker --version"\n      install_url: "https://docs.docker.com/get-docker/"\n      optional: false\n---\n\n# docker-deploy\n\nYou deploy applications using Docker. Steps:\n\n1. Analyze the project structure to determine the appropriate base image\n2. Generate a multi-stage Dockerfile optimized for size\n3. Create docker-compose.yml if multiple services are needed\n4. Build and test locally\n5. Push to registry if credentials are available',
     'Initial version', '00000000-0000-0000-0000-000000000010', 'approved', 'skill-md',
     ARRAY['deploy', 'dockerize', 'containerize'], 1200),
    ('00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000103',
     '1.0.0',
     E'---\nname: git-workflow\nversion: 1.0.0\nschema: skill-md\nframework: cursor\ntags: [git, workflow, automation]\ndescription: "Manages complex git workflows."\ntriggers: ["git workflow", "merge conflict", "branch strategy"]\nestimated_tokens: 800\nrequirements:\n  tools: [bash]\n  software:\n    - name: git\n      check_command: "git --version"\n      optional: false\n---\n\n# git-workflow\n\nYou manage git workflows. Capabilities:\n\n1. Create and manage feature branches following naming conventions\n2. Resolve merge conflicts by understanding both sides of changes\n3. Write descriptive PR/commit messages\n4. Suggest branch protection and review policies',
     'Initial version', '00000000-0000-0000-0000-000000000010', 'approved', 'skill-md',
     ARRAY['git workflow', 'merge conflict', 'branch strategy'], 800),
    ('00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000104',
     '2.0.0',
     E'---\nname: skillhub-discovery\nversion: 2.0.0\nschema: skill-md\nframework: gstack\ntags: [skillhub, discovery, search, meta]\ndescription: "The AI SkillHub Discovery Skill. Enables agents to search, install, rate, and contribute skills from the SkillHub registry."\ntriggers: ["find a skill", "search skillhub", "I need a tool for", "skillhub"]\ncompatible_models: [claude-3-5-sonnet, claude-opus-4, gpt-4o, gemini-2.5-pro]\nestimated_tokens: 500\nrequirements:\n  tools: [web_fetch, bash]\n---\n\n# SkillHub Discovery\n\nYou have access to SkillHub, the AI skill registry. Use it whenever you:\n- Need specialized knowledge for a complex task\n- Want to see if someone has solved a similar problem\n\n## API Reference\n\nBase URL: `https://skillhub.koolkassanmsk.top`\nToken: Read from the `SKILLHUB_TOKEN` environment variable.\nAll requests use `Authorization: Bearer $SKILLHUB_TOKEN`.\n\n### Search Skills\n```\nGET /v1/skills?q=<search_terms>&framework=<framework>&sort=<rating|installs|new>&limit=10\n```\n\n### Get Skill Content\n```\nGET /v1/skills/<namespace>/<name>/content\n```\n\n### Rate a Skill (always rate after use)\n```\nPOST /v1/skills/<namespace>/<name>/ratings\n{"score": 8, "outcome": "success", "task_type": "...", "model_used": "..."}\n```\n\n### Submit a Skill\n```\nPOST /v1/skills\n{"content": "---\\nname: ...\\n---\\n# Instructions...", "visibility": "public"}\n```\n\nBefore submitting: remove ALL personal info, API keys, and make instructions generic.',
     'v2 with full API reference', '00000000-0000-0000-0000-000000000010', 'approved', 'skill-md',
     ARRAY['find a skill', 'search skillhub', 'skillhub'], 500)
ON CONFLICT (skill_id, version) DO NOTHING;
