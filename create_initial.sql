CREATE TABLE `logs` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `user` varchar(100) NOT NULL DEFAULT '',
  `user_id` int(11) DEFAULT NULL,
  `project` varchar(100) NOT NULL DEFAULT '',
  `project_id` int(11) DEFAULT NULL,
  `commit_hash` varchar(100) NOT NULL DEFAULT '',
  `version` float DEFAULT NULL,
  `comment` varchar(100) NOT NULL DEFAULT '',
  `results` text NOT NULL,
  `created_on` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_on` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=latin1;
