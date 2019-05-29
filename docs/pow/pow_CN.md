# Soteria挖矿概述

*以其他语言阅读本文件: [English](pow.md)*

本文旨在概述目前在挖矿系统中涉及的算法和流程，适合没有相关背景知识的读者。在这个文档中，我们会先介绍一下图论中的环状结构和Soteria挖矿算法的基础：Cuckoo Cycle算法，之后我们会详细讨论该算法在Soteria挖矿中的特定修改和加强，进而对Soteria挖矿系统进行一个全面的介绍。

请注意，Soteria目前正在积极开发中，任何和所有这些技术细节我们都保留修改的权利，所有信息以最后发布的版本为准。此外，Soteria PoW在很大程度上依赖于Grin的算法。有关Grin的更多信息，请参阅: [Grin's PoW](https://github.com/mimblewimble/grin/blob/master/doc/pow/pow.md)
