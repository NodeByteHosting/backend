-- ============================================================================
-- CAREERS SCHEMA - Job Positions & Applications
-- ============================================================================

-- Job Positions (open positions on careers page)
CREATE TABLE IF NOT EXISTS job_positions (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    
    -- Job details
    department TEXT NOT NULL,
    "employmentType" TEXT NOT NULL, -- full_time, part_time, contract, internship
    location TEXT NOT NULL,
    "isRemote" BOOLEAN DEFAULT false,
    "salaryMin" DECIMAL(10, 2),
    "salaryMax" DECIMAL(10, 2),
    "salaryCurrency" TEXT DEFAULT 'USD',
    
    -- Job specifications
    "requiredSkills" TEXT[] DEFAULT '{}',
    "niceToHaveSkills" TEXT[] DEFAULT '{}',
    "yearsOfExperience" INTEGER,
    
    -- Content
    "shortDescription" TEXT,
    requirements TEXT,
    benefits TEXT,
    "aboutRole" TEXT,
    
    -- Status and visibility
    status TEXT DEFAULT 'draft', -- draft, published, closed, archived
    "isActive" BOOLEAN DEFAULT true,
    
    -- Metadata
    "createdById" TEXT REFERENCES users(id) ON DELETE SET NULL,
    
    "publishedAt" TIMESTAMP,
    "closedAt" TIMESTAMP,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_positions_slug ON job_positions(slug);
CREATE INDEX IF NOT EXISTS idx_job_positions_department ON job_positions(department);
CREATE INDEX IF NOT EXISTS idx_job_positions_employment_type ON job_positions("employmentType");
CREATE INDEX IF NOT EXISTS idx_job_positions_status ON job_positions(status);
CREATE INDEX IF NOT EXISTS idx_job_positions_is_active ON job_positions("isActive");
CREATE INDEX IF NOT EXISTS idx_job_positions_created_at ON job_positions("createdAt");

-- Job Applications (applications from candidates)
CREATE TABLE IF NOT EXISTS job_applications (
    id TEXT PRIMARY KEY,
    "jobPositionId" TEXT NOT NULL REFERENCES job_positions(id) ON DELETE CASCADE,
    
    -- Applicant info
    "firstName" TEXT NOT NULL,
    "lastName" TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    
    -- Application details
    "resumeUrl" TEXT,
    "portfolioUrl" TEXT,
    "linkedinUrl" TEXT,
    "githubUrl" TEXT,
    
    -- Cover letter and additional info
    "coverLetter" TEXT,
    "additionalInfo" JSONB DEFAULT '{}', -- custom fields, answers to screening questions, etc.
    
    -- Status tracking
    status TEXT DEFAULT 'new', -- new, reviewing, shortlisted, rejected, offered, hired, withdrawn
    "ratingScore" DECIMAL(3, 1), -- 0-5 star rating
    notes TEXT, -- internal notes for hiring team
    
    -- Metadata
    "appliedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "reviewedAt" TIMESTAMP,
    "reviewedById" TEXT REFERENCES users(id) ON DELETE SET NULL,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_applications_job_position_id ON job_applications("jobPositionId");
CREATE INDEX IF NOT EXISTS idx_job_applications_email ON job_applications(email);
CREATE INDEX IF NOT EXISTS idx_job_applications_status ON job_applications(status);
CREATE INDEX IF NOT EXISTS idx_job_applications_applied_at ON job_applications("appliedAt");
CREATE INDEX IF NOT EXISTS idx_job_applications_job_status ON job_applications("jobPositionId", status);

-- Job Application Activity (track application status changes and interactions)
CREATE TABLE IF NOT EXISTS job_application_activity (
    id TEXT PRIMARY KEY,
    "applicationId" TEXT NOT NULL REFERENCES job_applications(id) ON DELETE CASCADE,
    
    -- Activity log
    "activityType" TEXT NOT NULL, -- status_change, note_added, email_sent, interview_scheduled, etc.
    description TEXT,
    "oldStatus" TEXT,
    "newStatus" TEXT,
    
    -- Who performed the action
    "performedById" TEXT REFERENCES users(id) ON DELETE SET NULL,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_application_activity_application_id ON job_application_activity("applicationId");
CREATE INDEX IF NOT EXISTS idx_job_application_activity_type ON job_application_activity("activityType");
CREATE INDEX IF NOT EXISTS idx_job_application_activity_created_at ON job_application_activity("createdAt");
