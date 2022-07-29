#   Copyright (c) 2022-present, Adil Alper DALKIRAN
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
#   ==============================================================================

from abc import ABC, abstractmethod
import string

from inventa import Inventa, ServiceDescriptor

class BaseInferenceTask(ABC):
    def __init__(self, inventa: Inventa, SelfDescriptor: ServiceDescriptor):
        self.inventa = inventa
        self.SelfDescriptor = SelfDescriptor

    @abstractmethod
    def get_task_id(self) -> string:
        pass

    @abstractmethod
    def get_task_name(self) -> string:
        pass

    @abstractmethod
    async def run(self):
        pass
