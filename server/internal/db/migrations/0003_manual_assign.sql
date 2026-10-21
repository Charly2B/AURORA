-- 0.1.2 : marque les fichiers rattachés à la main (Administration › Identification). L'indexation ne les
-- reclasse jamais automatiquement.
ALTER TABLE media_files ADD COLUMN manual INTEGER NOT NULL DEFAULT 0;
