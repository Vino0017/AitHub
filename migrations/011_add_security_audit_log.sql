-- 011_add_security_audit_log.sql
-- 添加安全审计日志表

CREATE TABLE IF NOT EXISTS security_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    revision_id UUID NOT NULL REFERENCES revisions(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    issues JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 索引优化查询
CREATE INDEX idx_security_audit_log_revision ON security_audit_log(revision_id);
CREATE INDEX idx_security_audit_log_skill ON security_audit_log(skill_id);
CREATE INDEX idx_security_audit_log_event_type ON security_audit_log(event_type);
CREATE INDEX idx_security_audit_log_created_at ON security_audit_log(created_at DESC);

-- 用于分析攻击模式
CREATE INDEX idx_security_audit_log_event_created ON security_audit_log(event_type, created_at DESC);
